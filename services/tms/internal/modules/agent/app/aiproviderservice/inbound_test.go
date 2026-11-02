package aiproviderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// novaFor reports whether Nova runs, for the tenant it was asked about.
type novaFor struct {
	llmports.TaskRunner

	ready  bool
	tenant pagination.TenantInfo
}

func (n *novaFor) Available(_ context.Context, tenant pagination.TenantInfo) (bool, error) {
	n.tenant = tenant
	return n.ready, nil
}

func TestReadsInboundMail_WhenNovaRunsForTheTenant(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	for _, ready := range []bool{true, false} {
		nova := &novaFor{ready: ready}
		readiness := NewInboundReadiness(InboundReadinessParams{Nova: nova})

		reads, err := readiness.ReadsInboundMail(t.Context(), tenant)

		require.NoError(t, err)
		assert.Equal(t, ready, reads, "Nova reads inbound mail")
		assert.Equal(t, tenant, nova.tenant)
	}
}
