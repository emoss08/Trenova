package documentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/usageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/mocks/contentscanportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	storageutil "github.com/emoss08/trenova/shared/testutil/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type storageLevelGate struct {
	state *managedusage.State
	level int64
}

func (g *storageLevelGate) Decide(
	_ context.Context,
	_ managedscope.Scope,
	meter platformcatalog.MeterKey,
	requested int64,
) (managedusage.Decision, error) {
	if meter == platformcatalog.MeterDocumentStorageBytes {
		return managedusage.DecideAt(g.state, meter, g.level, requested), nil
	}
	return managedusage.Decide(g.state, meter, requested), nil
}

func (g *storageLevelGate) State(context.Context, managedscope.Scope) (*managedusage.State, error) {
	return g.state, nil
}

func TestUpload_AtTheStorageCeilingIsRefusedBeforeAnyByteIsStored(t *testing.T) {
	t.Parallel()

	svc, store := newScannedService(t, contentscanportsmock.NewRefusingGate(nil))
	svc.usageProvider = usageservice.NewManagedUsageProvider(&storageLevelGate{
		level: 249_999_999_000,
		state: &managedusage.State{
			WindowStart: 1,
			WindowEnd:   4_000_000_000,
			Overage:     true,
			Meters: []managedusage.Meter{{
				Meter:     platformcatalog.MeterDocumentStorageBytes,
				Unit:      managedusage.UnitBytes,
				Quantity:  249_999_999_000,
				Allowance: 25_000_000_000,
				Ceiling:   250_000_000_000,
			}},
		},
	}, managedscope.NewResolver(false))

	result, err := svc.Upload(t.Context(), &documentports.UploadRequest{
		TenantInfo: scannedTenant(),
		File: storageutil.NewMockFileHeader(
			"bol.pdf",
			append([]byte("%PDF-1.7\n"), make([]byte, 2_000)...),
			"application/pdf",
		),
		ResourceID:   "res_123",
		ResourceType: "trailer",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	var quota *errortypes.QuotaExceededError
	require.ErrorAs(t, err, &quota, "an upload that would cross the ceiling is refused")
	assert.Contains(t, quota.Message, "document storage is full at {0}")
	assert.Equal(t, []any{"250 GB"}, quota.Args)
	assert.Empty(t, store.Keys(), "no object is written for a refused upload")
}
