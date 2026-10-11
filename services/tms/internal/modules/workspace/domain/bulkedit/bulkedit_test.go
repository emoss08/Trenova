package bulkedit

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func validEdit() *BulkEdit {
	return &BulkEdit{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		Resource:       "customer",
		Field:          "status",
		Value:          "Inactive",
		Selection:      Selection{IDs: []string{"cus_1", "cus_2"}},
	}
}

func fieldsOf(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}
	return fields
}

func TestBulkEdit_Validate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*BulkEdit)
		field  string
	}{
		{name: "accepts chosen rows", mutate: func(*BulkEdit) {}},
		{
			name: "accepts every row matching a filter",
			mutate: func(b *BulkEdit) {
				b.Selection = Selection{Filter: &SelectionFilter{Query: "acme"}}
			},
		},
		{
			name:   "refuses no rows at all",
			mutate: func(b *BulkEdit) { b.Selection = Selection{} },
			field:  "selection",
		},
		{
			name: "refuses both chosen rows and a filter",
			mutate: func(b *BulkEdit) {
				b.Selection.Filter = &SelectionFilter{}
			},
			field: "selection",
		},
		{
			name: "refuses more rows than one edit can change",
			mutate: func(b *BulkEdit) {
				b.Selection.IDs = make([]string, MaxSelectedIDs+1)
				for i := range b.Selection.IDs {
					b.Selection.IDs[i] = "cus_x"
				}
			},
			field: "selection.ids",
		},
		{
			name:   "refuses a blank row ID",
			mutate: func(b *BulkEdit) { b.Selection.IDs = []string{"cus_1", " "} },
			field:  "selection.ids[1]",
		},
		{name: "requires a field", mutate: func(b *BulkEdit) { b.Field = "" }, field: "field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edit := validEdit()
			tt.mutate(edit)

			multiErr := errortypes.NewMultiError()
			edit.Validate(multiErr)

			if tt.field == "" {
				assert.False(t, multiErr.HasErrors(), "unexpected: %v", fieldsOf(multiErr))
				return
			}
			assert.Contains(t, fieldsOf(multiErr), tt.field)
		})
	}
}

func TestBulkEdit_CanUndo(t *testing.T) {
	completed := int64(1_000_000)
	edit := validEdit()
	edit.Status = StatusDone
	edit.ChangedCount = 3
	edit.CompletedAt = &completed

	assert.True(t, edit.CanUndo(completed+60))
	assert.True(t, edit.CanUndo(completed+UndoWindowSeconds))
	assert.False(t, edit.CanUndo(completed+UndoWindowSeconds+1), "past the window")

	edit.ChangedCount = 0
	assert.False(t, edit.CanUndo(completed+60), "nothing changed")

	edit.ChangedCount = 3
	edit.Status = StatusUndone
	assert.False(t, edit.CanUndo(completed+60), "already undone")
}
