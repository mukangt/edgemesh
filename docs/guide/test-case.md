# Test Case

All the test cases in this chapter can find the corresponding files in the directory [examples](https://github.com/kubeedge/edgemesh/tree/main/examples).

## Prepare

- **Step 1**: Deploy EdgeMesh

Please refer to [Getting Started](https://edgemesh.netlify.app/guide/) to deploy EdgeMesh

- **Step 2**: Deploy Test Pods

```shell
$ kubectl apply -f examples/test-pod.yaml
pod/alpine-test created
pod/websocket-test created
```

## HTTP

Deploy a HTTP container application and relevant service

```shell
$ kubectl apply -f examples/hostname.yaml
deployment.apps/hostname-edge created
service/hostname-svc created
```

Enter the test pod and use `curl` to access the service

```shell
$ kubectl exec -it alpine-test -- sh
(in the container environment)
/ # curl hostname-svc:12345
hostname-edge-5c75d56dc4-rq57t
```

## HTTPS

Deploy a HTTPS container application and relevant service

```shell
$ ./examples/nginx-https/tools.sh install
...
Getting Private key
Getting CA Private Key
secret/nginxsecret created
configmap/nginxconfigmap created
deployment.apps/nginx-https created
service/nginx-https created
create https example success!
```

Enter the test pod and use `curl` to access the service

```shell
$ kubectl exec -it alpine-test -- sh
(in the container environment)
/ # curl -k --cert client.crt --key client.key https://nginx-https
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
...

(you can also use external domain name to access related service)
/ # curl --cacert rootCA.crt --cert client.crt --key client.key https://my-nginx.com
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
...
```

::: details
examples/nginx-https/tools.sh script function:
1. Generate self-signed root certificate, server, client certificates and private keys
2. Create nginx-https related secret, configmap, deployment and service
3. Copy the certificate and private key to alpine-test
4. Write the mapping between IP and domain name (my-nginx.com) to the /etc/hosts file in alpine-test

Note: Use the cleanup command of the tools.sh script to clear all the resources created above and restore the modification to alpine-test
:::

## TCP

Deploy a TCP container application and relevant service

```shell
$ kubectl apply -f examples/tcp-echo-service.yaml
deployment.apps/tcp-echo-deployment created
service/tcp-echo-service created
```

Enter the test pod and use `telnet` to access the service

```shell
$ kubectl exec -it alpine-test -- sh
(in the container environment)
/ # telnet tcp-echo-service 2701
Welcome, you are connected to node ke-edge1.
Running on Pod tcp-echo-deployment-66457b769-7zgqb.
In namespace default.
With IP address 172.17.0.2.
Service default.
```

## Websocket

Deploy a websocket container application and relevant service

```shell
$ kubectl apply -f examples/websocket.yaml
deployment.apps/ws-edge created
service/ws-svc created
```

Enter the test pod and use websocket `client` to access the service

```shell
$ kubectl exec -it websocket-test -- sh
(in the container environment)
/ # ./client --addr ws-svc:12348
connecting to ws://ws-svc.default:12348/echo
recv: 2021-12-02 03:42:20.191695384 +0000 UTC m=+1.004526202
recv: 2021-12-02 03:42:21.191724176 +0000 UTC m=+2.004554995
recv: 2021-12-02 03:42:22.191725321 +0000 UTC m=+3.004556159
```

## UDP

Deploy a UDP container application and relevant service

```shell
$ kubectl apply -f examples/hostname-udp.yaml
deployment.apps/hostname-edge created
service/hostname-udp-svc created
```

Enter the test pod and use `nc` to access the service

```shell
$ kubectl exec -it alpine-test -- sh
(in the container environment)
/ # nc -u hostname-udp-svc 12345
hostname-edge-5cd47b65d5-8zg27
```

## Load Balance

Deploy a container application and related services configured with a `random` load balancing strategy

```shell
$ kubectl apply -f examples/hostname-lb-random.yaml
deployment.apps/hostname-lb-edge created
service/hostname-lb-svc created
destinationrule.networking.istio.io/hostname-lb-svc created
```

:::tip
EdgeMesh uses the loadBalancer property in DestinationRule to select different load balancing strategies. While using DestinationRule, the name of the DestinationRule must be equal to the name of the corresponding Service. EdgeMesh will determine the DestinationRule in the same namespace according to the name of the Service
:::

Enter the test pod and use `curl` multiple times to access the service, you will see that multiple hostname-edge are randomly accessed

```shell
$ kubectl exec -it alpine-test -- sh
(in the container environment)
/ # curl hostname-lb-svc:12345
hostname-lb-edge-7898fff5f9-w82nw
/ # curl hostname-lb-svc:12345
hostname-lb-edge-7898fff5f9-xjp86
/ # curl hostname-lb-svc:12345
hostname-lb-edge-7898fff5f9-xjp86
/ # curl hostname-lb-svc:12345
hostname-lb-edge-7898fff5f9-iq39z
```

## Node-Select Load Balance

The `NodeSelectPolicy` allows callers to steer individual HTTP requests to a pod running on a **specific edge node** by setting the `X-EdgeMesh-Target-Node` request header — without requiring a separate Service per node.

### Enable on a Service

Add the annotation `edgemesh.kubeedge.io/node-select: "true"` to the Service. No DestinationRule is needed.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: hostname-lb-svc
  annotations:
    edgemesh.kubeedge.io/node-select: "true"
    # Optional: fall back to a random endpoint when the target node has no pods.
    # Omit or set to "false" (default) to return an error instead.
    edgemesh.kubeedge.io/node-select-fallback: "true"
spec:
  selector:
    app: hostname-lb-edge
  ports:
    - name: http-0
      port: 12345
      protocol: TCP
      targetPort: 9376
```

Deploy the example:

```shell
$ kubectl apply -f examples/hostname-lb-nodeselect.yaml
deployment.apps/hostname-lb-edge created
service/hostname-lb-svc created
```

:::tip
The annotation is hot-reloaded — adding or removing it takes effect immediately without restarting EdgeMesh.
:::

### Send a request to a specific node

Pass the `X-EdgeMesh-Target-Node` header with the target node name:

```shell
$ kubectl exec -it alpine-test -- sh
# Route to edge-node-1
/ # curl -H "X-EdgeMesh-Target-Node: edge-node-1" hostname-lb-svc:12345
hostname-lb-edge-7898fff5f9-xjp86

# Route to edge-node-2
/ # curl -H "X-EdgeMesh-Target-Node: edge-node-2" hostname-lb-svc:12345
hostname-lb-edge-7898fff5f9-w82nw

# No header → random endpoint (same as RANDOM policy)
/ # curl hostname-lb-svc:12345
hostname-lb-edge-7898fff5f9-iq39z
```

:::tip
The header key supports two forms:
- `X-EdgeMesh-Target-Node` (canonical)
- `x-edgemesh-target-node` (HTTP/2 all-lowercase, e.g. forwarded by Envoy)
:::

### Behavior summary

| Scenario | `node-select-fallback: "false"` (default) | `node-select-fallback: "true"` |
|---|---|---|
| Header present, node has pods | Routes to a pod on that node | Routes to a pod on that node |
| Header present, node has **no** pods | Returns an error | Falls back to a random endpoint |
| Header **absent** | Random endpoint | Random endpoint |

## Cross-Edge-Cloud :star:

The busybox-edge in the edgezone can access the tcp-echo-cloud on the cloud, and the busybox-cloud in the cloudzone can access the tcp-echo-edge on the edge

**Deploy**

```shell
$ kubectl apply -f examples/cloudzone.yaml
namespace/cloudzone created
deployment.apps/tcp-echo-cloud created
service/tcp-echo-cloud-svc created
deployment.apps/busybox-sleep-cloud created
```

```
$ kubectl apply -f examples/edgezone.yaml
namespace/edgezone created
deployment.apps/tcp-echo-edge created
service/tcp-echo-edge-svc created
deployment.apps/busybox-sleep-edge created
```

**Cloud access edge**

```shell
$ BUSYBOX_POD=$(kubectl get all -n cloudzone | grep pod/busybox | awk '{print $1}')
$ kubectl -n cloudzone exec $BUSYBOX_POD -c busybox -i -t -- sh
$ telnet tcp-echo-edge-svc.edgezone 2701
Welcome, you are connected to node ke-edge1.
Running on Pod tcp-echo-edge.
In namespace edgezone.
With IP address 172.17.0.2.
Service default.
Hello Edge, I am Cloud.
Hello Edge, I am Cloud.
```

**Edge access cloud**

At the edge node, use `telnet` to access the service

```shell
$ BUSYBOX_CID=$(docker ps | grep k8s_busybox_busybox-sleep-edge | awk '{print $1}')
$ docker exec -it $BUSYBOX_CID sh
$ telnet tcp-echo-cloud-svc.cloudzone 2701
Welcome, you are connected to node k8s-master.
Running on Pod tcp-echo-cloud.
In namespace cloudzone.
With IP address 10.244.0.8.
Service default.
Hello Cloud, I am Edge.
Hello Cloud, I am Edge.
```
