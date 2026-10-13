package triageshipmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRecorder struct {
	record *detentionservice.DisputeRecord
	asked  []*detentionservice.DisputeRecordParams
}

func (f *fakeRecorder) DisputeRecord(
	_ context.Context,
	p *detentionservice.DisputeRecordParams,
) (*detentionservice.DisputeRecord, error) {
	f.asked = append(f.asked, p)

	return f.record, nil
}

func TestDetentionEvidenceAsksForTheDisputedOccurrencesRecord(t *testing.T) {
	t.Parallel()

	small := &detention.DetentionOccurrence{
		ID:             pulid.MustNew("dto_"),
		BillableAmount: decimal.RequireFromString("50"),
	}
	disputed := &detention.DetentionOccurrence{
		ID:             pulid.MustNew("dto_"),
		BillableAmount: decimal.RequireFromString("1200"),
	}
	documentID := pulid.MustNew("doc_")
	recorder := &fakeRecorder{record: &detentionservice.DisputeRecord{
		DocumentID: documentID,
		FileName:   "detention-record-SEED-DET-008.pdf",
		Rendered:   true,
	}}
	service := &Service{
		l:           zap.NewNop(),
		occurrences: &fakeOccurrences{occurrences: []*detention.DetentionOccurrence{small, disputed}},
		records:     recorder,
	}
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	evidence, err := service.DetentionEvidence(t.Context(), &inboundmessageports.TriageDetentionEvidenceRequest{
		TenantInfo: tenantInfo,
		ShipmentID: pulid.MustNew("shp_"),
	})

	require.NoError(t, err)
	require.NotNil(t, evidence)
	assert.Equal(t, documentID, evidence.DocumentID)
	assert.True(t, evidence.Rendered)
	require.Len(t, recorder.asked, 1)
	assert.Equal(t, disputed.ID, recorder.asked[0].Occurrence.ID)
	assert.Equal(t, tenantInfo, recorder.asked[0].TenantInfo)
}

func TestDetentionEvidenceWithoutAnOccurrenceIsNothing(t *testing.T) {
	t.Parallel()

	recorder := &fakeRecorder{}
	service := &Service{
		l:           zap.NewNop(),
		occurrences: &fakeOccurrences{},
		records:     recorder,
	}

	evidence, err := service.DetentionEvidence(t.Context(), &inboundmessageports.TriageDetentionEvidenceRequest{
		ShipmentID: pulid.MustNew("shp_"),
	})

	require.NoError(t, err)
	assert.Nil(t, evidence)
	assert.Empty(t, recorder.asked)
}
