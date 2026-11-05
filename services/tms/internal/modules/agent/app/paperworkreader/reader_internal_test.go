package paperworkreader

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNovaReadsOnlyShipmentPaperworkItCanSee(t *testing.T) {
	t.Parallel()

	paperwork := func(fileType string, facts *document.DeliveryFacts) *document.Document {
		return &document.Document{
			ProcessingProfile: document.ProcessingProfileShipmentPaperwork,
			FileType:          fileType,
			DeliveryFacts:     facts,
		}
	}

	assert.True(t, readable(paperwork("application/pdf", nil)))
	assert.True(t, readable(paperwork("image/jpeg", nil)))
	assert.False(t, readable(paperwork("text/csv", nil)), "a file the model cannot read is skipped")
	assert.False(
		t,
		readable(
			paperwork(
				"application/pdf",
				&document.DeliveryFacts{Source: document.FactsSourcePerson},
			),
		),
		"facts a person recorded are never overwritten",
	)
	assert.False(t, readable(&document.Document{
		ProcessingProfile: document.ProcessingProfileNone,
		FileType:          "application/pdf",
	}))
}

func TestTheSchemaAsksForEveryFactAndTheDocumentKind(t *testing.T) {
	t.Parallel()

	got := schema()
	assert.Contains(t, got["required"], "documentKind")
	assert.Contains(t, got["required"], "signedBy")
	assert.Equal(t, false, got["additionalProperties"])
}

type documents struct {
	documentports.DocumentRepository

	doc      *document.Document
	recorded []*documentports.RecordDeliveryFactsRequest
}

func (d *documents) GetByID(
	context.Context,
	documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	return d.doc, nil
}

func (d *documents) RecordDeliveryFacts(
	_ context.Context,
	req *documentports.RecordDeliveryFactsRequest,
) (*document.Document, error) {
	d.recorded = append(d.recorded, req)

	return d.doc, nil
}

type nova struct {
	text string
	err  error
}

func (n nova) Available(context.Context, pagination.TenantInfo) (bool, error) {
	return n.err == nil, nil
}

func (n nova) Run(
	_ context.Context,
	task *llmports.NovaTask,
	out any,
) (*llmports.StructuredCompletionResult, error) {
	if n.err != nil {
		return nil, n.err
	}

	return &llmports.StructuredCompletionResult{
		Text: n.text,
	}, llmports.DecodeAnswer(
		task.Name,
		n.text,
		out,
	)
}

func pendingPaperwork() *document.Document {
	return &document.Document{
		ID:                pulid.MustNew("doc_"),
		ProcessingProfile: document.ProcessingProfileShipmentPaperwork,
		FileType:          "application/pdf",
	}
}

func TestRead_WritesOnlyTheFactsAndNeverOverAPersons(t *testing.T) {
	t.Parallel()

	repo := &documents{doc: pendingPaperwork()}
	reader := &Reader{
		harness:      nova{text: `{"signedBy":"Dana Cole","documentKind":"ProofOfDelivery"}`},
		documents:    repo,
		accessorials: noAccessorials{},
		l:            zap.NewNop(),
	}

	require.NoError(t, reader.Read(t.Context(), repo.doc.ID, pagination.TenantInfo{}, pulid.Nil))

	require.Len(t, repo.recorded, 1)
	assert.Equal(t, "Dana Cole", repo.recorded[0].Facts.SignedBy)
	assert.True(t, repo.recorded[0].KeepPerson)
	assert.Nil(t, repo.recorded[0].Version)
}

func TestRead_LeavesTheDocumentWithoutRetryingWhenNovaCannotRead(t *testing.T) {
	t.Parallel()

	for name, harness := range map[string]nova{
		"no provider":        {err: llmports.ErrNoProviderConfigured},
		"outside its schema": {text: "Sure, here it is."},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repo := &documents{doc: pendingPaperwork()}
			reader := &Reader{
				harness: harness, documents: repo, accessorials: noAccessorials{}, l: zap.NewNop(),
			}

			require.NoError(
				t,
				reader.Read(t.Context(), repo.doc.ID, pagination.TenantInfo{}, pulid.Nil),
				"an error would retry a call that cannot do better",
			)
			assert.Empty(t, repo.recorded)
		})
	}
}

type noAccessorials struct {
	accessorialchargeports.AccessorialChargeRepository
}

func (noAccessorials) ListCatalog(
	context.Context,
	pagination.TenantInfo,
) ([]*accessorialcharge.AccessorialCharge, error) {
	return nil, nil
}
