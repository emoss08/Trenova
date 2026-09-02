package redis

import (
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/emoss08/gtc/internal/core/domain"
	"go.uber.org/zap"
)

func newTestJSONSink(t *testing.T) (*JSONSink, *miniredis.Miniredis) {
	t.Helper()

	server := miniredis.RunT(t)
	sink, err := NewJSONSink("redis://"+server.Addr(), zap.NewNop())
	if err != nil {
		t.Fatalf("NewJSONSink returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := sink.Shutdown(t.Context()); err != nil {
			t.Errorf("Shutdown returned error: %v", err)
		}
	})

	return sink, server
}

func customerCacheProjection() domain.Projection {
	return domain.Projection{
		Name:         "customer-cache",
		SourceSchema: "public",
		SourceTable:  "customers",
		PrimaryKeys:  []string{"id", "business_unit_id", "organization_id"},
		Destination: domain.Destination{
			Kind:        domain.DestinationRedisJSON,
			KeyTemplate: `cache:customers:{{ value "organization_id" .New .Old }}:{{ value "business_unit_id" .New .Old }}:{{ value "id" .New .Old }}`,
		},
	}
}

func TestJSONSinkTruncateDeletesOnlyProjectionKeys(t *testing.T) {
	t.Parallel()

	sink, server := newTestJSONSink(t)

	projectionKeys := []string{
		"cache:customers:org_1:bu_1:cus_1",
		"cache:customers:org_1:bu_1:cus_2",
		"cache:customers:org_2:bu_9:cus_3",
	}
	unrelatedKeys := []string{
		"cache:workers:org_1:bu_1:wrk_1",
		"cache:organization:org_1",
		"cache:customersextra",
	}
	for _, key := range append(append([]string{}, projectionKeys...), unrelatedKeys...) {
		if err := server.Set(key, "{}"); err != nil {
			t.Fatalf("seed key %s: %v", key, err)
		}
	}

	err := sink.Write(t.Context(), customerCacheProjection(), domain.SourceRecord{
		Operation: domain.OperationTruncate,
		Schema:    "public",
		Table:     "customers",
	})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	for _, key := range projectionKeys {
		if server.Exists(key) {
			t.Fatalf("expected projection key %s to be deleted", key)
		}
	}
	for _, key := range unrelatedKeys {
		if !server.Exists(key) {
			t.Fatalf("expected unrelated key %s to survive the truncate", key)
		}
	}
}

func TestJSONSinkTruncateDeletesBeyondOneBatch(t *testing.T) {
	t.Parallel()

	sink, server := newTestJSONSink(t)

	keyCount := truncateDeleteBatch*2 + 7
	for idx := range keyCount {
		key := fmt.Sprintf("cache:customers:org_1:bu_1:cus_%d", idx)
		if err := server.Set(key, "{}"); err != nil {
			t.Fatalf("seed key %s: %v", key, err)
		}
	}

	err := sink.Write(t.Context(), customerCacheProjection(), domain.SourceRecord{
		Operation: domain.OperationTruncate,
		Schema:    "public",
		Table:     "customers",
	})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if remaining := server.Keys(); len(remaining) != 0 {
		t.Fatalf("expected all keys deleted, %d remain: %v", len(remaining), remaining[:min(len(remaining), 5)])
	}
}

func TestJSONSinkTruncateFailsWhenPatternHasNoLiteralAnchor(t *testing.T) {
	t.Parallel()

	sink, server := newTestJSONSink(t)

	if err := server.Set("unrelated", "{}"); err != nil {
		t.Fatalf("seed key: %v", err)
	}

	projection := customerCacheProjection()
	projection.Destination.KeyTemplate = `{{ value "id" .New .Old }}`

	err := sink.Write(t.Context(), projection, domain.SourceRecord{
		Operation: domain.OperationTruncate,
		Schema:    "public",
		Table:     "customers",
	})
	if err == nil {
		t.Fatalf("expected error for all-placeholder key template")
	}

	if !server.Exists("unrelated") {
		t.Fatalf("expected unrelated key to survive a refused truncate")
	}
}
