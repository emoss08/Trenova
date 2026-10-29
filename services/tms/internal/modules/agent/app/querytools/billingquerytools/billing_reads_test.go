package billingquerytools

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/gettool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqllisttool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// invoiceCards answers DeskInvoiceDetail by the id each read sends, and
// counts how many reads run at once.
type invoiceCards struct {
	mu       sync.Mutex
	cards    map[string]*graphqlreadports.GraphQLReadResult
	sent     []string
	inFlight atomic.Int32
	most     atomic.Int32
}

func (c *invoiceCards) Read(
	_ context.Context,
	req *graphqlreadports.GraphQLReadRequest,
) (*graphqlreadports.GraphQLReadResult, error) {
	now := c.inFlight.Add(1)
	defer c.inFlight.Add(-1)
	for {
		most := c.most.Load()
		if now <= most || c.most.CompareAndSwap(most, now) {
			break
		}
	}
	time.Sleep(time.Millisecond)

	id, _ := req.Variables[graphqlread.IDVariable].(string)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, id)
	if card, ok := c.cards[id]; ok {
		return card, nil
	}

	return &graphqlreadports.GraphQLReadResult{
		Errors: []graphqlreadports.GraphQLReadError{{
			Code: graphqlreadports.GraphQLCodeNotFound, Message: "invoice not found",
		}},
	}, nil
}

func invoiceCard(id, number string, lines int) *graphqlreadports.GraphQLReadResult {
	rows := make([]any, 0, lines)
	for idx := range lines {
		rows = append(rows, map[string]any{
			"id": fmt.Sprintf("invl_%d", idx), "lineNumber": idx + 1, "description": "Linehaul",
		})
	}

	return &graphqlreadports.GraphQLReadResult{
		Data: map[string]any{"invoice": map[string]any{
			"id": id, "number": number, "status": "Posted", "memo": nil, "lines": rows,
		}},
		Withheld: []string{"invoice.totalAmount", "invoice.lines.amount"},
	}
}

func invoiceGetTool(t *testing.T, reader graphqlreadports.GraphQLReader) *gettool.GraphQLTool {
	t.Helper()

	spec := InvoiceGetSpec()
	tool, err := gettool.NewGraphQL(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	return tool
}

func TestInvoiceGet_OneIDReadsTheInvoiceWithEveryLineByID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("inv_").String()
	reader := &invoiceCards{cards: map[string]*graphqlreadports.GraphQLReadResult{
		id: invoiceCard(id, "INV-1042", 3),
	}}

	data, err := invoiceGetTool(
		t,
		reader,
	).Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		"invoiceIds": []any{id, id},
	}))
	require.NoError(t, err)

	record, ok := data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "INV-1042", record["number"])
	assert.Equal(t, 3, record[fieldLineCount])
	lines, _ := record["lines"].([]any)
	require.Len(t, lines, 3)
	assert.Equal(t, "invl_0", lines[0].(map[string]any)["id"])
	assert.NotContains(t, record, fieldLinesTruncated)
	assert.Equal(t, []string{"totalAmount", "lines.amount"}, record[graphqlread.OutcomeWithheld])
	assert.Equal(t, []string{id}, reader.sent, "a repeated id is read once")
}

func TestInvoiceGet_SeveralIDsAreReadOneAtATimeAtMostTenAtOnce(t *testing.T) {
	t.Parallel()

	cards := make(map[string]*graphqlreadports.GraphQLReadResult)
	ids := make([]any, 0, 30)
	for idx := range 30 {
		id := pulid.MustNew("inv_").String()
		cards[id] = invoiceCard(id, fmt.Sprintf("INV-%d", idx), 40)
		ids = append(ids, id)
	}
	stranger := pulid.MustNew("inv_").String()
	ids = slices.Insert(ids, 1, any(stranger))
	reader := &invoiceCards{cards: cards}

	data, err := invoiceGetTool(
		t,
		reader,
	).Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		"invoiceIds": ids,
	}))
	require.NoError(t, err)

	outcome, ok := data.(map[string]any)
	require.True(t, ok)
	items, _ := outcome[graphqlread.OutcomeItems].([]map[string]any)
	require.Len(t, items, 30)
	assert.Equal(t, "INV-0", items[0]["number"], "invoices come back in the order named")
	assert.Equal(t, "INV-1", items[1]["number"])
	for _, item := range items {
		lines, _ := item["lines"].([]any)
		assert.Len(
			t,
			lines,
			invoiceLinesShownEach,
			"an invoice read beside others shows fewer lines",
		)
		assert.Equal(t, true, item[fieldLinesTruncated])
		assert.Equal(t, 40, item[fieldLineCount])
		assert.NotContains(t, item, graphqlread.OutcomeWithheld, "what is withheld is named once")
	}
	assert.Contains(t, outcome[graphqlread.OutcomeNote], stranger)
	assert.Equal(t, []string{"lines.amount", "totalAmount"}, outcome[graphqlread.OutcomeWithheld])
	assert.Len(t, reader.sent, 31, "one read per id")
	assert.LessOrEqual(t, reader.most.Load(), int32(gettool.GraphQLAtOnce))
	assert.Greater(t, reader.most.Load(), int32(1), "the reads run concurrently")
}

func TestInvoiceGet_ARefusedReadRefusesTheCall(t *testing.T) {
	t.Parallel()

	allowed := pulid.MustNew("inv_").String()
	refused := pulid.MustNew("inv_").String()
	reader := &invoiceCards{cards: map[string]*graphqlreadports.GraphQLReadResult{
		allowed: invoiceCard(allowed, "INV-1", 1),
		refused: {Errors: []graphqlreadports.GraphQLReadError{{
			Code: graphqlreadports.GraphQLCodeForbidden, Message: "not allowed",
		}}},
	}}

	_, err := invoiceGetTool(t, reader).Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		"invoiceIds": []any{allowed, refused},
	}))
	var denied *graphqlreadports.ReadRefusedError
	require.ErrorAs(t, err, &denied)
	assert.Equal(t, permission.ResourceInvoice, denied.Resource)
}

func TestInvoiceGet_RefusesWhatIsNotAnIDAndTooManyIDs(t *testing.T) {
	t.Parallel()

	reader := &invoiceCards{}
	tool := invoiceGetTool(t, reader)

	_, err := tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{"invoiceIds": []any{"INV-1042"}}),
	)
	require.Error(t, err)

	many := make([]any, 0, invoiceGetBatch+1)
	for range invoiceGetBatch + 1 {
		many = append(many, pulid.MustNew("inv_").String())
	}
	_, err = tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{"invoiceIds": many}))
	require.Error(t, err)
	assert.Empty(t, reader.sent, "nothing is read for a refused call")

	properties, _ := tool.ParamSchema()["properties"].(map[string]any)
	ids, _ := properties["invoiceIds"].(map[string]any)
	assert.Equal(t, "array", ids["type"])
	assert.Equal(t, invoiceGetBatch, ids["maxItems"])
}

func TestInvoiceDisputes_SendsTheInvoicesTheCallNames(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("inv_").String(), pulid.MustNew("inv_").String()
	reader := &querytooltest.KeyedReader{Answers: map[string]*graphqlreadports.GraphQLReadResult{
		"DeskInvoiceDisputes": {Data: map[string]any{"invoiceDisputes": []any{
			map[string]any{"id": "idsp_1", "invoiceId": first, "status": "Open"},
		}}, Withheld: []string{"invoiceDisputes.notes"}},
	}}
	var spec readspec.List
	for _, candidate := range ReadSpecs() {
		if candidate.Name == toolInvoiceListDisputes {
			spec = candidate
		}
	}
	tool, err := graphqllisttool.NewRead(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	data, err := tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{
		"invoiceIds": []any{first, second},
	}))
	require.NoError(t, err)
	sent := reader.Sent("DeskInvoiceDisputes")
	require.NotNil(t, sent)
	assert.Equal(t, []any{first, second}, sent.Variables["invoiceIds"])
	outcome, _ := data.(map[string]any)
	assert.Equal(t, 1, outcome[graphqlread.OutcomeCount])
	assert.Equal(t, []string{"notes"}, outcome[graphqlread.OutcomeWithheld])

	_, err = tool.Query(
		t.Context(),
		querytooltest.KeyedCall(map[string]any{"invoiceIds": []any{"INV-1"}}),
	)
	require.Error(t, err)
	_, err = tool.Query(t.Context(), querytooltest.KeyedCall(map[string]any{}))
	require.Error(t, err, "the invoices are required")
}

// What each billing get promises, the operation it reads selects.
func TestBillingGets_TheOperationSelectsWhatTheSummaryPromises(t *testing.T) {
	t.Parallel()

	for tool, want := range map[string][]string{
		toolInvoiceGet: {
			"lines", "settlementStatus", "disputeStatus", "sendStatus", "openBalance",
			"totalAmount", "memo", "voidReason",
		},
		toolInvoiceGetAdjustment: {
			"kind", "status", "approvalStatus", "creditMemoInvoiceId", "replacementInvoiceId",
			"supportingDocumentIds", "lines", "reason",
		},
	} {
		var names []string
		for _, field := range graphqlread.Reads[tool].Fields {
			if field.Selected {
				names = append(names, field.Name)
			}
		}
		for _, field := range want {
			assert.True(t, slices.Contains(names, field), "%s selects %s", tool, field)
		}
	}
}
