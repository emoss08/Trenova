package invoiceadjustmentservice

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetentionWaiversSurviveTheAdjustmentsMetadata(t *testing.T) {
	t.Parallel()

	waiver := detentionports.DetentionWaiver{
		OccurrenceID: pulid.MustNew("dto_"),
		Reason:       detention.WaiverReasonOther,
		Note:         "Decided in Triage",
		RequestedBy:  pulid.MustNew("usr_"),
	}
	attachmentID := pulid.MustNew("doc_")
	metadata := adjustmentMetadata(&invoiceadjustmentports.InvoiceAdjustmentRequest{
		AttachmentIDs:    []pulid.ID{attachmentID},
		DetentionWaivers: []detentionports.DetentionWaiver{waiver},
	})

	encoded, err := sonic.Marshal(metadata)
	require.NoError(t, err)
	stored := map[string]any{}
	require.NoError(t, sonic.Unmarshal(encoded, &stored))

	waivers, err := detentionWaiversOf(&invoiceadjustment.InvoiceAdjustment{Metadata: stored})
	require.NoError(t, err)
	assert.Equal(t, []detentionports.DetentionWaiver{waiver}, waivers)
	assert.Contains(t, stored, metadataAttachmentIDs)
}

func TestAnAdjustmentWithoutWaiversCarriesNone(t *testing.T) {
	t.Parallel()

	metadata := adjustmentMetadata(&invoiceadjustmentports.InvoiceAdjustmentRequest{})
	assert.NotContains(t, metadata, metadataDetentionWaivers)

	waivers, err := detentionWaiversOf(&invoiceadjustment.InvoiceAdjustment{Metadata: metadata})
	require.NoError(t, err)
	assert.Empty(t, waivers)

	_, err = detentionWaiversOf(&invoiceadjustment.InvoiceAdjustment{
		Metadata: map[string]any{metadataDetentionWaivers: "not a list"},
	})
	require.Error(t, err)
}
