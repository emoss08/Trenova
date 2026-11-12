package tooldispatch

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

const (
	askedCustomerA = "cus_01JW3K18A8TPC7T7SPX7745V1H"
	askedCustomerB = "cus_01JSV4G0Q80KNYKQ254QWT5T0V"
)

type askedLabeler struct {
	labels agentruntimeports.RecordLabels
	err    error
	reads  int
}

func (l *askedLabeler) Labels(
	context.Context,
	pagination.TenantInfo,
	map[permission.Resource][]pulid.ID,
) (agentruntimeports.RecordLabels, error) {
	l.reads++

	return l.labels, l.err
}

func narrowedCall() (*llmports.ToolCall, map[string]any) {
	call := &llmports.ToolCall{
		ID:   "call_1",
		Name: "shipment_list",
		Arguments: map[string]any{"filters": []any{
			map[string]any{"field": "customerId", "operator": "in"},
			map[string]any{"field": "status", "operator": "in"},
		}},
	}
	document := map[string]any{
		"count": float64(3),
		"searchedFor": []any{
			"customerId one of " + askedCustomerA + ", " + askedCustomerB,
			"status one of New, PartiallyAssigned",
		},
	}

	return call, document
}

func TestAsked_NamesTheRecordsAReadWasNarrowedBy(t *testing.T) {
	t.Parallel()

	labeler := &askedLabeler{labels: agentruntimeports.RecordLabels{
		permission.ResourceCustomer: {
			pulid.ID(askedCustomerA): "Granite Textiles",
			pulid.ID(askedCustomerB): "Granite Brands",
		},
	}}
	dispatcher := &Dispatcher{logger: zap.NewNop(), labels: labeler}
	call, document := narrowedCall()

	asked := dispatcher.asked(t.Context(), &agentruntimeports.RunRequest{}, call, document)

	assert.Equal(t,
		"Customer Granite Textiles, Granite Brands · Status New, Partially assigned",
		asked,
	)
	assert.Equal(t, 1, labeler.reads)
}

func TestAsked_AFailedReadCountsTheRecordsAndShowsNoID(t *testing.T) {
	t.Parallel()

	dispatcher := &Dispatcher{
		logger: zap.NewNop(),
		labels: &askedLabeler{err: errors.New("read failed")},
	}
	call, document := narrowedCall()

	asked := dispatcher.asked(t.Context(), &agentruntimeports.RunRequest{}, call, document)

	assert.Equal(t, "Customer 2 selected · Status New, Partially assigned", asked)
	assert.False(t, strings.Contains(asked, "cus_"))
}

func TestAsked_AResultThatIsNotANarrowedListSaysNothing(t *testing.T) {
	t.Parallel()

	labeler := &askedLabeler{}
	dispatcher := &Dispatcher{logger: zap.NewNop(), labels: labeler}
	call := &llmports.ToolCall{ID: "call_1", Name: "shipment_get"}

	assert.Empty(t, dispatcher.asked(
		t.Context(), &agentruntimeports.RunRequest{}, call,
		map[string]any{"proNumber": "S1"},
	))
	assert.Empty(t, dispatcher.asked(
		t.Context(), &agentruntimeports.RunRequest{}, call,
		map[string]any{"searchedFor": []any{"no filters: the most recent shipments"}},
	))
	assert.Zero(t, labeler.reads)
}
