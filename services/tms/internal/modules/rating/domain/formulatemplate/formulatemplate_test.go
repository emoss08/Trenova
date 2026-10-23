package formulatemplate

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

const perMileSource = `tariff "Per Mile"

input rate: Money/mi
input minimumCharge: Money = $0

charge "Per Mile" (quantity: shipment.distance, rate: rate)
minimum minimumCharge
`

func validTemplate() *FormulaTemplate {
	return &FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Per Mile",
		Source:         perMileSource,
		Status:         StatusDraft,
	}
}

func fieldsOf(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}
	return fields
}

func TestFormulaTemplate_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*FormulaTemplate)
		fields []string
	}{
		{name: "a named template with source passes", mutate: func(*FormulaTemplate) {}},
		{
			name:   "a blank name is refused",
			mutate: func(ft *FormulaTemplate) { ft.Name = "" },
			fields: []string{"name"},
		},
		{
			name:   "a name past 100 characters is refused",
			mutate: func(ft *FormulaTemplate) { ft.Name = strings.Repeat("a", 101) },
			fields: []string{"name"},
		},
		{
			name:   "an empty source is refused",
			mutate: func(ft *FormulaTemplate) { ft.Source = "" },
			fields: []string{"source"},
		},
		{
			name: "a source past 64 KiB is refused",
			mutate: func(ft *FormulaTemplate) {
				ft.Source = perMileSource + strings.Repeat("/", tarifftypes.MaxSourceLength)
			},
			fields: []string{"source"},
		},
		{
			name:   "an unknown status is refused",
			mutate: func(ft *FormulaTemplate) { ft.Status = Status("Published") },
			fields: []string{"status"},
		},
		{
			name: "a description past 2,000 characters is refused",
			mutate: func(ft *FormulaTemplate) {
				ft.Description = strings.Repeat("é", MaxDescriptionLength+1)
			},
			fields: []string{"description"},
		},
		{
			name: "metadata past 16 KiB is refused",
			mutate: func(ft *FormulaTemplate) {
				ft.Metadata = map[string]any{"notes": strings.Repeat("x", MaxMetadataBytes)}
			},
			fields: []string{"metadata"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entity := validTemplate()
			tt.mutate(entity)
			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)

			assert.ElementsMatch(t, tt.fields, fieldsOf(multiErr))
		})
	}
}

func TestFormulaTemplate_HasMaterialChange(t *testing.T) {
	t.Parallel()

	libraryID := pulid.MustNew("fl_")
	base := validTemplate()
	base.Imports = []tarifftypes.Import{
		{Path: "org.accessorials", LibraryID: libraryID, VersionNumber: 2},
	}

	t.Run("nothing but the name changed", func(t *testing.T) {
		t.Parallel()
		other := *base
		other.Name = "Renamed"
		other.Description = "New words"
		assert.False(t, other.HasMaterialChange(base))
	})

	t.Run("the source changed", func(t *testing.T) {
		t.Parallel()
		other := *base
		other.Source = strings.Replace(base.Source, "$0", "$50", 1)
		assert.True(t, other.HasMaterialChange(base))
	})

	t.Run("an import resolved to a newer library version", func(t *testing.T) {
		t.Parallel()
		other := *base
		other.Imports = []tarifftypes.Import{
			{Path: "org.accessorials", LibraryID: libraryID, VersionNumber: 3},
		}
		assert.True(t, other.HasMaterialChange(base))
	})

	t.Run("compared with nothing", func(t *testing.T) {
		t.Parallel()
		assert.True(t, base.HasMaterialChange(nil))
	})
}

func TestFormulaTemplate_ApplyVersionCarriesTheVersionsContent(t *testing.T) {
	t.Parallel()

	template := validTemplate()
	template.CurrentVersionNumber = 5
	version := &FormulaTemplateVersion{
		VersionNumber: 3,
		Name:          "Old name",
		Description:   "Old words",
		Source:        "tariff \"Old\"\n\ncharge \"Flat\" = $10\n",
		Inputs: []tarifftypes.InputDecl{
			{Name: "rate", Type: "Money", Descriptor: rules.RawJSON(`{"kind":"money"}`), Required: true},
		},
		Imports:   []tarifftypes.Import{{Path: "org.accessorials", VersionNumber: 1}},
		TestCount: 4,
	}

	applied := template.ApplyVersion(version)
	assert.Equal(t, version.Source, applied.Source)
	assert.Equal(t, version.Inputs, applied.Inputs)
	assert.Equal(t, version.Imports, applied.Imports)
	assert.EqualValues(t, 4, applied.TestCount)
	assert.EqualValues(t, 3, applied.CurrentVersionNumber)
	assert.Equal(t, template.Name, applied.Name, "ApplyVersion keeps the row's identity")
	assert.Equal(t, perMileSource, template.Source, "the receiver is not modified")

	full := template.ApplyVersionFull(version)
	assert.Equal(t, "Old name", full.Name)
	assert.Equal(t, "Old words", full.Description)
}

func TestNewVersionFromTemplate_SnapshotsTheContent(t *testing.T) {
	t.Parallel()

	template := validTemplate()
	template.Inputs = []tarifftypes.InputDecl{{Name: "rate", Required: true}}
	template.Imports = []tarifftypes.Import{{Path: "org.accessorials", VersionNumber: 1}}
	template.TestCount = 2
	template.Metadata = map[string]any{"owner": "pricing"}
	userID := pulid.MustNew("usr_")

	version := NewVersionFromTemplate(template, 7, userID, "Tightened", nil)

	assert.Equal(t, template.ID, version.TemplateID)
	assert.EqualValues(t, 7, version.VersionNumber)
	assert.Equal(t, template.Source, version.Source)
	assert.Equal(t, template.Inputs, version.Inputs)
	assert.Equal(t, template.Imports, version.Imports)
	assert.EqualValues(t, 2, version.TestCount)
	assert.Equal(t, template.Status, version.Status)
	assert.Equal(t, "Tightened", version.ChangeMessage)
	assert.Equal(t, userID, version.CreatedByID)
	assert.Equal(t, template.Metadata, version.Metadata)
}

func TestBeforeAppendModel_StoresEmptyCollectionsNotNull(t *testing.T) {
	t.Parallel()

	template := &FormulaTemplate{Name: "Flat", Source: "tariff \"Flat\"\n"}
	require.NoError(t, template.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)))
	assert.True(t, template.ID.IsNotNil())
	assert.NotNil(t, template.Inputs)
	assert.NotNil(t, template.Imports)
	assert.NotZero(t, template.CreatedAt)

	version := &FormulaTemplateVersion{}
	require.NoError(t, version.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)))
	assert.True(t, version.ID.IsNotNil())
	assert.NotNil(t, version.Inputs)
	assert.NotNil(t, version.Imports)
}

func TestFormulaTemplateVersion_ValidateRequiresSource(t *testing.T) {
	t.Parallel()

	version := &FormulaTemplateVersion{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		TemplateID:     pulid.MustNew("ft_"),
		VersionNumber:  1,
		Name:           "Per Mile",
		Status:         StatusDraft,
		CreatedByID:    pulid.MustNew("usr_"),
	}
	multiErr := errortypes.NewMultiError()
	version.Validate(multiErr)
	assert.Equal(t, []string{"source"}, fieldsOf(multiErr))
}

func TestStatusFromString(t *testing.T) {
	t.Parallel()

	for _, status := range []Status{StatusActive, StatusInactive, StatusDraft, StatusInReview} {
		parsed, err := StatusFromString(status.String())
		require.NoError(t, err)
		assert.Equal(t, status, parsed)
		assert.True(t, parsed.IsValid())
	}
	_, err := StatusFromString("Published")
	assert.Error(t, err)
}

func TestCanTransition_ArchivedTemplatesReenterReview(t *testing.T) {
	t.Parallel()

	assert.True(t, CanTransition(StatusInactive, StatusInReview))
	assert.False(t, CanTransition(StatusInactive, StatusActive))
	assert.True(t, CanTransition(StatusDraft, StatusInReview))
	assert.False(t, CanTransition(StatusDraft, StatusActive))
}

func TestVersionTag_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, VersionTagStable.IsValid())
	assert.False(t, VersionTag("Golden").IsValid())
}
