package assistantservice

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type bunchingRepo struct {
	stubArtifactRepo

	deleted  []pulid.ID
	listedBy []repositories.ListArtifactsByToolCallsRequest
}

func (r *bunchingRepo) current() []*assistantartifact.Artifact {
	latest := make(map[pulid.ID]*assistantartifact.Artifact, len(r.upserts))
	order := make([]pulid.ID, 0, len(r.upserts))
	for _, artifact := range r.upserts {
		if _, seen := latest[artifact.ID]; !seen {
			order = append(order, artifact.ID)
		}
		latest[artifact.ID] = artifact
	}

	out := make([]*assistantartifact.Artifact, 0, len(order))
	for _, id := range order {
		if !slices.Contains(r.deleted, id) {
			out = append(out, latest[id])
		}
	}

	return out
}

func (r *bunchingRepo) ListByToolCalls(
	_ context.Context,
	req *repositories.ListArtifactsByToolCallsRequest,
) ([]*assistantartifact.Artifact, error) {
	r.listedBy = append(r.listedBy, *req)
	out := make([]*assistantartifact.Artifact, 0, len(req.CallIDs))
	for _, artifact := range r.current() {
		if slices.Contains(req.CallIDs, artifact.SourceToolCallID) {
			out = append(out, artifact)
		}
	}

	return out, nil
}

func (r *bunchingRepo) Delete(_ context.Context, req *repositories.DeleteArtifactsRequest) error {
	r.deleted = append(r.deleted, req.IDs...)

	return nil
}

func invoiceRead(callID string, number string, earlier ...string) serviceports.ToolObservation {
	return serviceports.ToolObservation{
		Call: serviceports.ToolCall{ID: callID, Name: "get_invoice"},
		Data: map[string]any{
			"id":          pulid.MustNew("inv_").String(),
			"number":      number,
			"status":      "Draft",
			"billToName":  "Acme Foods",
			"totalAmount": "2100.00",
		},
		Earlier: earlier,
	}
}

func bunchingRecorder(t *testing.T) (*artifactRecorder, *bunchingRepo, pagination.TenantInfo) {
	t.Helper()

	repo := &bunchingRepo{}
	svc := &Service{logger: zap.NewNop(), artifacts: repo}
	thread := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	return svc.newArtifactRecorder(t.Context(), thread, tenant, testActor(), nil), repo, tenant
}

func TestBunch_ASingleGetStillMakesItsCard(t *testing.T) {
	t.Parallel()

	recorder, repo, _ := bunchingRecorder(t)

	shown, err := recorder.observe(invoiceRead("call_1", "INV-1"))
	require.NoError(t, err)
	require.NotNil(t, shown)
	assert.Equal(t, string(assistantartifact.KindEntityCard), shown.Kind)
	assert.Empty(t, repo.listedBy, "a first read has nothing to bunch with")
}

func TestBunch_SeveralGetsOfOneToolInATurnBecomeOneTable(t *testing.T) {
	t.Parallel()

	first, repo, tenant := bunchingRecorder(t)
	_, err := first.observe(invoiceRead("call_1", "INV-1"))
	require.NoError(t, err)
	card := repo.current()[0]

	second := first
	shown, err := second.observe(invoiceRead("call_2", "INV-2", "call_1"))
	require.NoError(t, err)
	require.NotNil(t, shown)
	assert.Equal(t, string(assistantartifact.KindTableView), shown.Kind)
	assert.Equal(t, []pulid.ID{card.ID}, repo.deleted, "the first card is folded into the table")
	require.Len(t, repo.listedBy, 1)
	assert.Equal(t, tenant, repo.listedBy[0].TenantInfo)
	assert.Equal(t, []string{"call_1"}, repo.listedBy[0].CallIDs)

	_, err = second.observe(invoiceRead("call_3", "INV-3", "call_1", "call_2"))
	require.NoError(t, err)

	live := repo.current()
	require.Len(t, live, 1, "one table for the turn and tool, not a card per call")
	table := live[0]
	assert.Equal(t, assistantartifact.KindTableView, table.Kind)
	assert.Equal(t, "call_1", table.SourceToolCallID, "keyed by the turn's first call of the tool")
	assert.Equal(t, "Invoice (3)", table.Title)
	assert.Equal(t, true, table.Payload[payloadBunched])
	assert.Equal(t, []string{"call_1", "call_2", "call_3"}, table.Payload[payloadCalls])
	assert.Equal(t, "invoice", table.Payload["recordEntity"])

	rows := table.Payload[payloadRows].([]any)
	require.Len(t, rows, 3)
	numbers := make([]any, 0, 3)
	for _, row := range rows {
		record := row.(map[string]any)
		numbers = append(numbers, record["number"])
		assert.NotEmpty(t, record[recordIDKey], "each row opens its invoice")
	}
	assert.Equal(t, []any{"INV-1", "INV-2", "INV-3"}, numbers)
}

func TestBunch_AnEarlierCallWithNoCardKeepsTheNewCard(t *testing.T) {
	t.Parallel()

	recorder, repo, _ := bunchingRecorder(t)
	_, err := recorder.observe(invoiceRead("call_1", "INV-1"))
	require.NoError(t, err)

	other := invoiceRead("call_2", "INV-2", "call_9")
	shown, err := recorder.observe(other)
	require.NoError(t, err)
	assert.Equal(t, string(assistantartifact.KindEntityCard), shown.Kind)
	assert.Empty(t, repo.deleted)
}

func TestBunch_TheTurnKeepsTheTableNotTheCardsItReplaced(t *testing.T) {
	t.Parallel()

	recorder, repo, _ := bunchingRecorder(t)
	_, err := recorder.observe(invoiceRead("call_1", "INV-1"))
	require.NoError(t, err)
	card := repo.current()[0]
	_, err = recorder.observe(invoiceRead("call_2", "INV-2", "call_1"))
	require.NoError(t, err)
	table := repo.current()[0]

	finish, _, _ := bunchingRecorder(t)
	finish.adopt([]*assistantartifact.Artifact{card, table})

	require.Len(t, finish.recorded, 1)
	assert.Equal(t, table.ID, finish.recorded[0].ID)
}

func TestArtifactFromObservation_ABatchGetIsATable(t *testing.T) {
	t.Parallel()

	artifact := artifactFromObservation(observation("get_invoices", map[string]any{
		"count":   2,
		"columns": []any{"id", "number", "status"},
		"items": []any{
			map[string]any{"id": pulid.MustNew("inv_").String(), "number": "INV-1", "status": "Draft"},
			map[string]any{"id": pulid.MustNew("inv_").String(), "number": "INV-2", "status": "Draft"},
		},
	}))

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindTableView, artifact.Kind)
	assert.Equal(t, "invoices", artifact.Payload["entity"])
	assert.Len(t, artifact.Payload["rows"], 2)
}
