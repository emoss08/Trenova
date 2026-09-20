package httpsafe

import (
	"testing"
	"time"
)

// A streaming client must carry no whole-request deadline: http.Client.Timeout
// spans reading the body, so any value here is a wall-clock budget for the
// whole answer and severs a healthy stream partway through.
func TestNewStreamingClientWithPolicy_HasNoWholeRequestDeadline(t *testing.T) {
	t.Parallel()

	client := NewStreamingClientWithPolicy(Policy{ResponseHeaderTimeout: 3 * time.Second})

	if client.Timeout != 0 {
		t.Fatalf("streaming client carries a whole-request timeout of %s", client.Timeout)
	}
}

// Dropping the deadline must not drop the egress guard with it.
func TestNewStreamingClientWithPolicy_KeepsTheEgressGuard(t *testing.T) {
	t.Parallel()

	client := NewStreamingClientWithPolicy(Policy{})

	transport := client.Transport
	if transport == nil {
		t.Fatal("streaming client has no transport, so it has no egress guard")
	}
	if client.CheckRedirect == nil {
		t.Fatal("streaming client follows redirects, which escapes the guard")
	}
}
