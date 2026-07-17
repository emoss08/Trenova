package sidebarpreference

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldErrors(multiErr *errortypes.MultiError) map[string]errortypes.ErrorCode {
	fields := make(map[string]errortypes.ErrorCode, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields[err.Field] = err.Code
	}
	return fields
}

func TestDefaultDocument(t *testing.T) {
	t.Parallel()

	doc := DefaultDocument()

	require.Len(t, doc.Sections, 5)
	assert.Equal(t, SectionAttention, doc.Sections[0].Key)
	assert.Equal(t, SectionBrowse, doc.Sections[4].Key)
	for _, section := range doc.Sections {
		assert.False(t, section.Hidden)
	}

	assert.Len(t, doc.AttentionMetrics, len(AttentionMetricCatalog()))
	assert.Equal(
		t,
		[]string{"create-shipment", "create-worker", "create-location", "create-customer"},
		doc.QuickActionIDs,
	)
	assert.Equal(t, DefaultActivityPageSize, doc.Activity.PageSize)
	assert.True(t, doc.Activity.DefaultOpen)
	assert.Equal(t, DocumentSchemaVersion, doc.SchemaVersion)

	multiErr := errortypes.NewMultiError()
	doc.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}

func TestDocument_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(doc *Document)
		wantErr string
	}{
		{
			name:    "invalid schema version",
			mutate:  func(doc *Document) { doc.SchemaVersion = 2 },
			wantErr: "schemaVersion",
		},
		{
			name: "unknown section key",
			mutate: func(doc *Document) {
				doc.Sections[0].Key = "bogus"
			},
			wantErr: "sections[0].key",
		},
		{
			name: "duplicate section key",
			mutate: func(doc *Document) {
				doc.Sections[1].Key = doc.Sections[0].Key
			},
			wantErr: "sections[1].key",
		},
		{
			name: "browse section hidden",
			mutate: func(doc *Document) {
				doc.Sections[4].Hidden = true
			},
			wantErr: "sections[4].hidden",
		},
		{
			name: "unknown attention metric",
			mutate: func(doc *Document) {
				doc.AttentionMetrics[0] = "bogusMetric"
			},
			wantErr: "attentionMetrics[0]",
		},
		{
			name: "duplicate attention metric",
			mutate: func(doc *Document) {
				doc.AttentionMetrics[1] = doc.AttentionMetrics[0]
			},
			wantErr: "attentionMetrics[1]",
		},
		{
			name: "unknown quick action",
			mutate: func(doc *Document) {
				doc.QuickActionIDs[0] = "create-bogus"
			},
			wantErr: "quickActionIds[0]",
		},
		{
			name: "duplicate quick action",
			mutate: func(doc *Document) {
				doc.QuickActionIDs[1] = doc.QuickActionIDs[0]
			},
			wantErr: "quickActionIds[1]",
		},
		{
			name: "too many quick actions",
			mutate: func(doc *Document) {
				doc.QuickActionIDs = []string{
					"create-shipment",
					"create-worker",
					"create-location",
					"create-customer",
					"create-tractor",
					"create-trailer",
					"create-commodity",
				}
			},
			wantErr: "quickActionIds",
		},
		{
			name: "invalid activity page size",
			mutate: func(doc *Document) {
				doc.Activity.PageSize = 7
			},
			wantErr: "activity.pageSize",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := DefaultDocument()
			tt.mutate(doc)

			multiErr := errortypes.NewMultiError()
			doc.Validate(multiErr)

			require.True(t, multiErr.HasErrors())
			assert.Contains(t, fieldErrors(multiErr), tt.wantErr)
		})
	}
}

func TestDocument_Normalize(t *testing.T) {
	t.Parallel()

	t.Run("drops unknown entries and appends missing sections", func(t *testing.T) {
		t.Parallel()

		doc := &Document{
			SchemaVersion: DocumentSchemaVersion,
			Sections: []SectionPreference{
				{Key: "bogus"},
				{Key: SectionBrowse, Hidden: true},
				{Key: SectionActivity, Hidden: true},
				{Key: SectionActivity},
			},
			AttentionMetrics: []string{"bogusMetric", "serviceFailures", "serviceFailures"},
			QuickActionIDs:   []string{"create-bogus", "create-shipment", "create-shipment"},
			Activity:         ActivityPreference{PageSize: 7, DefaultOpen: true},
		}

		normalized := doc.Normalize()

		require.Len(t, normalized.Sections, 5)
		assert.Equal(t, SectionBrowse, normalized.Sections[0].Key)
		assert.False(t, normalized.Sections[0].Hidden)
		assert.Equal(t, SectionActivity, normalized.Sections[1].Key)
		assert.True(t, normalized.Sections[1].Hidden)
		assert.Equal(t, SectionAttention, normalized.Sections[2].Key)
		assert.Equal(t, SectionQuickActions, normalized.Sections[3].Key)
		assert.Equal(t, SectionFavorites, normalized.Sections[4].Key)

		assert.Equal(t, []string{"serviceFailures"}, normalized.AttentionMetrics)
		assert.Equal(t, []string{"create-shipment"}, normalized.QuickActionIDs)
		assert.Equal(t, DefaultActivityPageSize, normalized.Activity.PageSize)
		assert.True(t, normalized.Activity.DefaultOpen)
	})

	t.Run("caps quick actions at the maximum", func(t *testing.T) {
		t.Parallel()

		doc := DefaultDocument()
		doc.QuickActionIDs = []string{
			"create-shipment",
			"create-worker",
			"create-location",
			"create-customer",
			"create-tractor",
			"create-trailer",
			"create-commodity",
		}

		normalized := doc.Normalize()

		assert.Len(t, normalized.QuickActionIDs, MaxQuickActions)
		assert.NotContains(t, normalized.QuickActionIDs, "create-commodity")
	})
}
