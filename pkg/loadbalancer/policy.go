package loadbalancer

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"

	"github.com/buraksezer/consistent"
	istioapi "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/pkg/apis/config/v1alpha1"
)

const (
	RoundRobin     = "ROUND_ROBIN"
	Random         = "RANDOM"
	ConsistentHash = "CONSISTENT_HASH"

	HTTPHeader   = "HTTP_HEADER"
	UserSourceIP = "USER_SOURCE_IP"
)

type Policy interface {
	Name() string
	Update(oldDr, dr *istioapi.DestinationRule)
	Pick(endpoints []string, srcAddr net.Addr, tcpConn net.Conn, cliReq *http.Request) (string, *http.Request, error)
	Sync(endpoints []string)
	Release()
}

// RoundRobinPolicy is a default policy.
type RoundRobinPolicy struct {
}

func NewRoundRobinPolicy() *RoundRobinPolicy {
	return &RoundRobinPolicy{}
}

func (*RoundRobinPolicy) Name() string {
	return RoundRobin
}

func (*RoundRobinPolicy) Update(_, _ *istioapi.DestinationRule) {}

func (*RoundRobinPolicy) Pick(_ []string, _ net.Addr, _ net.Conn, cliReq *http.Request) (string, *http.Request, error) {
	// RoundRobinPolicy is an empty implementation and we won't use it,
	// the outer round-robin policy will be used next.
	return "", cliReq, fmt.Errorf("call RoundRobinPolicy is forbidden")
}

func (*RoundRobinPolicy) Sync(_ []string) {}

func (*RoundRobinPolicy) Release() {}

type RandomPolicy struct {
	lock sync.Mutex
}

func NewRandomPolicy() *RandomPolicy {
	return &RandomPolicy{}
}

func (rd *RandomPolicy) Name() string {
	return Random
}

func (rd *RandomPolicy) Update(_, _ *istioapi.DestinationRule) {}

func (rd *RandomPolicy) Pick(endpoints []string, _ net.Addr, _ net.Conn, cliReq *http.Request) (string, *http.Request, error) {
	rd.lock.Lock()
	k := rand.Int() % len(endpoints)
	rd.lock.Unlock()
	return endpoints[k], cliReq, nil
}

func (rd *RandomPolicy) Sync(_ []string) {}

func (rd *RandomPolicy) Release() {}

type ConsistentHashPolicy struct {
	Config   *v1alpha1.ConsistentHash
	lock     sync.Mutex
	hashRing *consistent.Consistent
	hashKey  HashKey
}

func NewConsistentHashPolicy(config *v1alpha1.ConsistentHash, dr *istioapi.DestinationRule, endpoints []string) *ConsistentHashPolicy {
	return &ConsistentHashPolicy{
		Config:   config,
		hashRing: newHashRing(config, endpoints),
		hashKey:  getConsistentHashKey(dr),
	}
}

func (ch *ConsistentHashPolicy) Name() string {
	return ConsistentHash
}

func (ch *ConsistentHashPolicy) Update(_, dr *istioapi.DestinationRule) {
	ch.lock.Lock()
	ch.hashKey = getConsistentHashKey(dr)
	ch.lock.Unlock()
}

func (ch *ConsistentHashPolicy) Pick(_ []string, srcAddr net.Addr, netConn net.Conn, cliReq *http.Request) (endpoint string, req *http.Request, err error) {
	ch.lock.Lock()
	defer ch.lock.Unlock()

	req = cliReq
	var keyValue string
	switch ch.hashKey.Type {
	case HTTPHeader:
		if req == nil {
			req, err = http.ReadRequest(bufio.NewReader(netConn))
			if err != nil {
				klog.Errorf("read http request err: %v", err)
				return "", nil, err
			}
		}
		keyValue = req.Header.Get(ch.hashKey.Key)
	case UserSourceIP:
		if srcAddr == nil && netConn != nil {
			srcAddr = netConn.RemoteAddr()
		}
		keyValue = srcAddr.String()
	default:
		klog.Errorf("Failed to get hash key value")
		keyValue = ""
	}
	klog.Infof("Get key value: %s", keyValue)
	member := ch.hashRing.LocateKey([]byte(keyValue))
	if member == nil {
		errMsg := fmt.Errorf("can't find a endpoint by given key: %s", keyValue)
		klog.Errorf("%v", errMsg)
		return "", req, errMsg
	}
	return member.String(), req, nil
}

func (ch *ConsistentHashPolicy) Sync(endpoints []string) {
	ch.lock.Lock()
	if ch.hashRing == nil {
		ch.hashRing = newHashRing(ch.Config, endpoints)
	} else {
		updateHashRing(ch.hashRing, endpoints)
	}
	ch.lock.Unlock()
}

func (ch *ConsistentHashPolicy) Release() {
	ch.lock.Lock()
	clearHashRing(ch.hashRing)
	ch.lock.Unlock()
}

const NodeSelect = "NODE_SELECT"

// NodeSelectPolicy routes each request to the node specified by the
// X-EdgeMesh-Target-Node HTTP request header. When the header is absent
// the request is forwarded to a randomly chosen endpoint.
//
// Activate on a Service with:
//
//	edgemesh.kubeedge.io/node-select: "true"
//
// Optional fallback when the target node has no endpoints:
//
//	edgemesh.kubeedge.io/node-select-fallback: "true"  (route to any node)
//	edgemesh.kubeedge.io/node-select-fallback: "false" (return error, default)
type NodeSelectPolicy struct {
	lock     sync.Mutex
	fallback bool // route to random endpoint when target node has no endpoints
}

func NewNodeSelectPolicy(fallback bool) *NodeSelectPolicy {
	return &NodeSelectPolicy{fallback: fallback}
}

func (n *NodeSelectPolicy) Name() string { return NodeSelect }

func (n *NodeSelectPolicy) Update(_, _ *istioapi.DestinationRule) {}

func (n *NodeSelectPolicy) Pick(endpoints []string, _ net.Addr, netConn net.Conn, cliReq *http.Request) (string, *http.Request, error) {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Try to parse the HTTP request to read the target-node header,
	// mirroring how ConsistentHashPolicy reads HTTP_HEADER keys.
	var err error
	if cliReq == nil && netConn != nil {
		cliReq, err = http.ReadRequest(bufio.NewReader(netConn))
		if err != nil {
			klog.Warningf("NodeSelectPolicy: failed to read HTTP request, routing randomly: %v", err)
			return endpoints[rand.Intn(len(endpoints))], nil, nil
		}
	}

	// No parseable HTTP request (raw TCP/UDP) or header absent → random endpoint.
	targetNode := ""
	if cliReq != nil {
		targetNode = cliReq.Header.Get("X-EdgeMesh-Target-Node")
	}
	if targetNode == "" {
		return endpoints[rand.Intn(len(endpoints))], cliReq, nil
	}

	// Collect endpoints that live on the requested node.
	// Endpoint format: "nodeName:podName:ip:port"
	var matched []string
	var availableNodes []string
	for _, ep := range endpoints {
		node, _, _, _, ok := parseEndpoint(ep)
		if !ok {
			continue
		}
		if node == targetNode {
			matched = append(matched, ep)
		} else {
			availableNodes = append(availableNodes, node)
		}
	}

	if len(matched) > 0 {
		// Multiple pods on same node (non-DaemonSet): pick randomly among them.
		return matched[rand.Intn(len(matched))], cliReq, nil
	}

	// Target node has no endpoints.
	if n.fallback {
		klog.Warningf("NodeSelectPolicy: node %q has no endpoints, falling back to random (available: %v)", targetNode, availableNodes)
		return endpoints[rand.Intn(len(endpoints))], cliReq, nil
	}
	return "", cliReq, fmt.Errorf("NodeSelectPolicy: no endpoint on node %q, available nodes: %v", targetNode, availableNodes)
}

func (n *NodeSelectPolicy) Sync(_ []string) {}

func (n *NodeSelectPolicy) Release() {}
