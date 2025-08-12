package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/bootstrap"
	client "github.com/emoss08/trenova/shared/edi/client/configclient"
	"github.com/emoss08/trenova/shared/pulid"
)

// NOTE: This is a lightweight scaffold. It assumes a test config enabling gRPC server
// and a seeded test database containing at least one edi profile. Adjust as needed.
func TestEDIConfigService_GetPartnerConfig(t *testing.T) {
	t.Skip("integration environment required; seed DB and enable grpc in test config")
	go func() { _ = bootstrap.Bootstrap() }()
	time.Sleep(2 * time.Second)

	ctx := context.Background()
	c, err := client.Dial(ctx, client.DialOptions{Address: ":9090", Insecure: true})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	// Replace with real IDs in seeded fixtures
	bu := pulid.MustNew("bu_").String()
	org := pulid.MustNew("org_").String()
	_, err = c.Get(ctx, "", bu, org, "default")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
}
