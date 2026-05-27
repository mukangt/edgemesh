package loadbalancer

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// makeEndpoints builds a slice of "nodeName:podName:ip:port" strings, one per node.
func makeEndpoints(nodes []string) []string {
	eps := make([]string, len(nodes))
	for i, n := range nodes {
		eps[i] = fmt.Sprintf("%s:pod-%d:10.0.0.%d:8080", n, i, i+1)
	}
	return eps
}

// reqWithNode creates an http.Request carrying the X-EdgeMesh-Target-Node header.
func reqWithNode(node string) *http.Request {
	req, _ := http.NewRequest("GET", "http://svc/", nil)
	req.Header.Set("X-EdgeMesh-Target-Node", node)
	return req
}

// endpoint returns the nodeName portion of the chosen endpoint string.
func endpointNode(ep string) string {
	parts := strings.SplitN(ep, ":", 2)
	return parts[0]
}

// ─── Pick with header present ───────────────────────────────────────────────

func TestNodeSelectPolicy_PickMatchingNode(t *testing.T) {
	policy := NewNodeSelectPolicy(false)
	endpoints := makeEndpoints([]string{"edge-node-1", "edge-node-2", "edge-node-3"})

	ep, req, err := policy.Pick(endpoints, nil, nil, reqWithNode("edge-node-2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req == nil {
		t.Fatal("returned request must not be nil")
	}
	if got := endpointNode(ep); got != "edge-node-2" {
		t.Errorf("expected endpoint on edge-node-2, got %q", got)
	}
}

// ─── Multiple pods on the same node ─────────────────────────────────────────

func TestNodeSelectPolicy_PickMultiPodOnTargetNode(t *testing.T) {
	policy := NewNodeSelectPolicy(false)
	// Two pods on edge-node-1
	endpoints := []string{
		"edge-node-1:pod-a:10.0.0.1:8080",
		"edge-node-1:pod-b:10.0.0.2:8080",
		"edge-node-2:pod-c:10.0.0.3:8080",
	}

	for i := 0; i < 20; i++ {
		ep, _, err := policy.Pick(endpoints, nil, nil, reqWithNode("edge-node-1"))
		if err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}
		if got := endpointNode(ep); got != "edge-node-1" {
			t.Errorf("iteration %d: expected edge-node-1, got %q", i, got)
		}
	}
}

// ─── No header → random endpoint (any node accepted) ────────────────────────

func TestNodeSelectPolicy_PickNoHeader(t *testing.T) {
	policy := NewNodeSelectPolicy(false)
	endpoints := makeEndpoints([]string{"edge-node-1", "edge-node-2"})

	req, _ := http.NewRequest("GET", "http://svc/", nil) // no target header
	ep, _, err := policy.Pick(endpoints, nil, nil, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep == "" {
		t.Fatal("expected a non-empty endpoint")
	}
}

// ─── Target node absent, fallback=false → error with available nodes ─────────

func TestNodeSelectPolicy_MissingNodeNoFallback(t *testing.T) {
	policy := NewNodeSelectPolicy(false)
	endpoints := makeEndpoints([]string{"edge-node-1", "edge-node-2"})

	_, _, err := policy.Pick(endpoints, nil, nil, reqWithNode("edge-node-99"))
	if err == nil {
		t.Fatal("expected an error when target node has no endpoints")
	}
	// Error message must mention the missing node and at least one available node.
	if !strings.Contains(err.Error(), "edge-node-99") {
		t.Errorf("error should mention the missing node, got: %v", err)
	}
	if !strings.Contains(err.Error(), "edge-node-1") {
		t.Errorf("error should list available nodes, got: %v", err)
	}
}

// ─── Target node absent, fallback=true → pick any endpoint ───────────────────

func TestNodeSelectPolicy_MissingNodeWithFallback(t *testing.T) {
	policy := NewNodeSelectPolicy(true)
	endpoints := makeEndpoints([]string{"edge-node-1", "edge-node-2"})

	ep, _, err := policy.Pick(endpoints, nil, nil, reqWithNode("edge-node-99"))
	if err != nil {
		t.Fatalf("expected no error with fallback=true, got: %v", err)
	}
	if ep == "" {
		t.Fatal("expected a non-empty endpoint when falling back")
	}
}

// ─── nil cliReq and nil netConn → random endpoint, no panic ─────────────────

func TestNodeSelectPolicy_NilInputs(t *testing.T) {
	policy := NewNodeSelectPolicy(false)
	endpoints := makeEndpoints([]string{"edge-node-1"})

	ep, _, err := policy.Pick(endpoints, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error with nil inputs: %v", err)
	}
	if ep == "" {
		t.Fatal("expected a non-empty endpoint")
	}
}

// ─── syncNodeSelectPolicy annotation hot-reload ──────────────────────────────

func TestSyncNodeSelectPolicy_EnableDisable(t *testing.T) {
	// NewNodeSelectPolicy must honour the fallback flag.
	p1 := NewNodeSelectPolicy(false)
	p2 := NewNodeSelectPolicy(true)
	if p1.fallback {
		t.Error("expected fallback=false")
	}
	if !p2.fallback {
		t.Error("expected fallback=true")
	}
	if p1.Name() != NodeSelect {
		t.Errorf("expected policy name %q, got %q", NodeSelect, p1.Name())
	}
	// Release and Sync must not panic.
	p1.Release()
	p1.Sync(nil)
}
