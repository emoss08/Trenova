package agentquerytoolservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
A list result reaches the Desk as JSON, and by then its field order is gone:
the pane receives a map, and a map scatters "pro number, customer, status"
into whatever order it hashes to. Reading the order off the type is the only
place it still exists, which is why it is read here and carried.
*/

type columnRow struct {
	ProNumber string `json:"proNumber"`
	Customer  string `json:"customer"`
	Status    string `json:"status"`
}

type columnBase struct {
	ID      string `json:"id"`
	Version int64  `json:"version"`
}

type embeddedRow struct {
	columnBase
	Code string `json:"code"`
}

type taggedRow struct {
	Kept    string `json:"kept,omitempty"`
	Dropped string `json:"-"`
	Bare    string
	hidden  string
}

func TestColumnsAreTheProjectionsOwnOrder(t *testing.T) {
	outcome := newSearchCriteria("shipments").result([]any{
		columnRow{ProNumber: "P1", Customer: "Acme", Status: "InTransit"},
		columnRow{ProNumber: "P2"},
	}, 2)

	assert.Equal(t, []string{"proNumber", "customer", "status"}, outcome.Columns)
}

// An embedded struct is fields on the row, not a cell holding an object, so
// the grid draws them as the columns they are.
func TestColumnsFlattenAnEmbeddedProjection(t *testing.T) {
	outcome := newSearchCriteria("workers").result([]any{
		embeddedRow{columnBase: columnBase{ID: "w1", Version: 3}, Code: "W-1"},
	}, 1)

	assert.Equal(t, []string{"id", "version", "code"}, outcome.Columns)
}

func TestColumnsFollowTheJSONTag(t *testing.T) {
	outcome := newSearchCriteria("things").result([]any{taggedRow{Kept: "yes"}}, 1)

	assert.Equal(t, []string{"kept", "Bare"}, outcome.Columns)
}

// Rows arrive as pointers from some projections and values from others.
func TestColumnsReadThroughAPointerRow(t *testing.T) {
	outcome := newSearchCriteria("shipments").result([]any{&columnRow{ProNumber: "P1"}}, 1)

	assert.Equal(t, []string{"proNumber", "customer", "status"}, outcome.Columns)
}

// Nothing matched is not a table with no columns; it is no table.
func TestAnEmptyResultReportsNoColumns(t *testing.T) {
	outcome := newSearchCriteria("shipments").result([]any{}, 0)

	assert.Empty(t, outcome.Columns)
	assert.NotEmpty(t, outcome.Note)
}
