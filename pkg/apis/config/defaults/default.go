package defaults

import (
	"github.com/libp2p/go-libp2p/core/protocol"
)

type RunningMode string
type LoadBalancerCaller string
type TunnelMode string
type DiscoveryType string
type ServiceFilterMode string

const (
	BaseDir                   = "/etc/edgemesh/"
	ConfigDir                 = BaseDir + "config/"
	ConfigFilePath            = "config-file"
	EdgeMeshAgentConfigName   = "edgemesh-agent.yaml"
	EdgeMeshGatewayConfigName = "edgemesh-gateway.yaml"

	EdgeDNSModuleName     = "EdgeDNS"
	EdgeProxyModuleName   = "EdgeProxy"
	EdgeGatewayModuleName = "EdgeGateway"
	EdgeTunnelModuleName  = "EdgeTunnel"
	EdgeCNIModuleName     = "EdgeCNI"

	BridgeDeviceName = "edgemesh0"
	BridgeDeviceIP   = "169.254.96.16"
	TunDeviceName    = "edge_tun0"

	TempKubeConfigPath = "kubeconfig"
	TempCorefilePath   = "Corefile"
	MetaServerAddress  = "http://127.0.0.1:10550"
	MetaServerCertDir  = BaseDir + "metaserver/"
	MetaServerCaFile   = MetaServerCertDir + "rootCA.crt"
	MetaServerCertFile = MetaServerCertDir + "server.crt"
	MetaServerKeyFile  = MetaServerCertDir + "server.key"

	EdgeMode   RunningMode = "EdgeMode"   // detected running on the edge
	CloudMode  RunningMode = "CloudMode"  // detected running on the cloud
	ManualMode RunningMode = "ManualMode" // detected that user manually configured kubeAPIConfig

	EmptyNodeName = "EMPTY_NODE_NAME"
	EmptyPodName  = "EMPTY_POD_NAME"

	// LabelEdgeMeshServiceProxyName indicates that an alternative service proxy will implement this Service.
	LabelEdgeMeshServiceProxyName = "service.edgemesh.kubeedge.io/service-proxy-name"

	// AnnotationNodeSelect enables dynamic per-request node routing via the X-EdgeMesh-Target-Node HTTP header.
	// Set this annotation on a Service to activate the NodeSelect load-balancer policy.
	// Example: edgemesh.kubeedge.io/node-select: "true"
	AnnotationNodeSelect = "edgemesh.kubeedge.io/node-select"

	// AnnotationNodeSelectFallback controls what happens when the requested node has no healthy endpoints.
	// "true"  → fall back to a randomly chosen endpoint on another node.
	// "false" (default) → return an error so the caller is aware the target node is unavailable.
	// Example: edgemesh.kubeedge.io/node-select-fallback: "true"
	AnnotationNodeSelectFallback = "edgemesh.kubeedge.io/node-select-fallback"

	// HeaderTargetNode is the HTTP request header clients use to specify the target edge node.
	// Example: X-EdgeMesh-Target-Node: edge-node-1
	HeaderTargetNode = "X-EdgeMesh-Target-Node"

	ProxyCaller   LoadBalancerCaller = "ProxyCaller"
	GatewayCaller LoadBalancerCaller = "GatewayCaller"

	ClientMode       TunnelMode = "ClientOnly"
	ServerClientMode TunnelMode = "ServerAndClient"

	Rendezvous = "EDGEMESH_PLAYGOUND"
	PSKPath    = BaseDir + "psk"

	// DiscoveryProtocol and ProxyProtocol maintained by EdgeMesh Author
	DiscoveryProtocol protocol.ID = "/libp2p/tunnel-discovery/1.0.0"
	ProxyProtocol     protocol.ID = "/libp2p/tunnel-proxy/1.0.0"
	CNIProtocol       protocol.ID = "/libp2p/tunnel-cni/1.0.0"

	MdnsDiscovery DiscoveryType = "MDNS"
	DhtDiscovery  DiscoveryType = "DHT"

	FilterIfLabelExistsMode        ServiceFilterMode = "FilterIfLabelExists"
	FilterIfLabelDoesNotExistsMode ServiceFilterMode = "FilterIfLabelDoesNotExists"

	TunnelBaseStreamIn      int = 10240
	TunnelBaseStreamOut     int = 10240
	TunnelPeerBaseStreamIn  int = 1024
	TunnelPeerBaseStreamOut int = 1024
)
