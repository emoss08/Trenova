package formulatemplate

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestFormulaTemplate_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entity  FormulaTemplate
		wantErr bool
	}{
		{
			name: "valid entity passes",
			entity: FormulaTemplate{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Name:           "Test Template",
				Expression:     "rate * weight",
				Type:           TemplateTypeFreightCharge,
				Status:         StatusActive,
			},
			wantErr: false,
		},
		{
			name: "name empty fails",
			entity: FormulaTemplate{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Name:           "",
				Expression:     "rate * weight",
				Type:           TemplateTypeFreightCharge,
				Status:         StatusActive,
			},
			wantErr: true,
		},
		{
			name: "name too long fails",
			entity: FormulaTemplate{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Name:           strings.Repeat("a", 101),
				Expression:     "rate * weight",
				Type:           TemplateTypeFreightCharge,
				Status:         StatusActive,
			},
			wantErr: true,
		},
		{
			name: "expression empty fails",
			entity: FormulaTemplate{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Name:           "Test Template",
				Expression:     "",
				Type:           TemplateTypeFreightCharge,
				Status:         StatusActive,
			},
			wantErr: true,
		},
		{
			name: "invalid type fails",
			entity: FormulaTemplate{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Name:           "Test Template",
				Expression:     "rate * weight",
				Type:           TemplateType("Invalid"),
				Status:         StatusActive,
			},
			wantErr: true,
		},
		{
			name: "invalid status fails",
			entity: FormulaTemplate{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Name:           "Test Template",
				Expression:     "rate * weight",
				Type:           TemplateTypeFreightCharge,
				Status:         Status("Invalid"),
			},
			wantErr: true,
		},
		{
			name: "accessorial charge type passes",
			entity: FormulaTemplate{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Name:           "Accessorial Template",
				Expression:     "flat_rate + surcharge",
				Type:           TemplateTypeAccessorialCharge,
				Status:         StatusDraft,
			},
			wantErr: false,
		},
		{
			name: "inactive status passes",
			entity: FormulaTemplate{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Name:           "Test Template",
				Expression:     "rate * weight",
				Type:           TemplateTypeFreightCharge,
				Status:         StatusInactive,
			},
			wantErr: false,
		},
		{
			name: "name at max length passes",
			entity: FormulaTemplate{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Name:           strings.Repeat("a", 100),
				Expression:     "rate * weight",
				Type:           TemplateTypeFreightCharge,
				Status:         StatusActive,
			},
			wantErr: false,
		},
		{
			name: "empty type fails",
			entity: FormulaTemplate{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Name:           "Test",
				Expression:     "x",
				Type:           "",
				Status:         StatusActive,
			},
			wantErr: true,
		},
		{
			name: "empty status fails",
			entity: FormulaTemplate{
				ID:             pulid.MustNew("ft_"),
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				Name:           "Test",
				Expression:     "x",
				Type:           TemplateTypeFreightCharge,
				Status:         "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiErr := errortypes.NewMultiError()
			tt.entity.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestFormulaTemplate_GetTableName(t *testing.T) {
	t.Parallel()

	ft := &FormulaTemplate{}
	assert.Equal(t, "formula_templates", ft.GetTableName())
}

func TestFormulaTemplate_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ft_")
	ft := &FormulaTemplate{ID: id}
	assert.Equal(t, id, ft.GetID())
}

func TestFormulaTemplate_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	ft := &FormulaTemplate{OrganizationID: orgID}
	assert.Equal(t, orgID, ft.GetOrganizationID())
}

func TestFormulaTemplate_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	ft := &FormulaTemplate{BusinessUnitID: buID}
	assert.Equal(t, buID, ft.GetBusinessUnitID())
}

func TestFormulaTemplate_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	ft := &FormulaTemplate{}
	config := ft.GetPostgresSearchConfig()

	assert.Equal(t, "ft", config.TableAlias)
	assert.True(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 4)
	assert.Equal(t, "name", config.SearchableFields[0].Name)
	assert.Equal(t, "description", config.SearchableFields[1].Name)
	assert.Equal(t, "type", config.SearchableFields[2].Name)
	assert.Equal(t, "status", config.SearchableFields[3].Name)
}

func TestFormulaTemplate_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		ft := &FormulaTemplate{}
		require.True(t, ft.ID.IsNil())

		err := ft.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, ft.ID.IsNil())
		assert.NotZero(t, ft.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("ft_")
		ft := &FormulaTemplate{ID: existingID}

		err := ft.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, ft.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		ft := &FormulaTemplate{}

		err := ft.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, ft.UpdatedAt)
	})

	t.Run("select does not modify fields", func(t *testing.T) {
		t.Parallel()

		ft := &FormulaTemplate{}

		err := ft.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, ft.ID.IsNil())
		assert.Zero(t, ft.CreatedAt)
		assert.Zero(t, ft.UpdatedAt)
	})
}

func TestFormulaTemplateVersion_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		ftv := &FormulaTemplateVersion{}
		require.True(t, ftv.ID.IsNil())

		err := ftv.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, ftv.ID.IsNil())
		assert.NotZero(t, ftv.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("ftv_")
		ftv := &FormulaTemplateVersion{ID: existingID}

		err := ftv.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, ftv.ID)
	})

	t.Run("update does not modify fields", func(t *testing.T) {
		t.Parallel()

		ftv := &FormulaTemplateVersion{}

		err := ftv.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.True(t, ftv.ID.IsNil())
		assert.Zero(t, ftv.CreatedAt)
	})

	t.Run("select does not modify fields", func(t *testing.T) {
		t.Parallel()

		ftv := &FormulaTemplateVersion{}

		err := ftv.BeforeAppendModel(t.Context(), (*bun.SelectQuery)(nil))
		require.NoError(t, err)

		assert.True(t, ftv.ID.IsNil())
		assert.Zero(t, ftv.CreatedAt)
	})
}

func TestFormulaTemplateVersion_GetTableName(t *testing.T) {
	t.Parallel()

	ftv := &FormulaTemplateVersion{}
	assert.Equal(t, "formula_template_versions", ftv.GetTableName())
}

func TestFormulaTemplateVersion_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ftv_")
	ftv := &FormulaTemplateVersion{ID: id}
	assert.Equal(t, id, ftv.GetID())
}

func TestFormulaTemplateVersion_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	ftv := &FormulaTemplateVersion{OrganizationID: orgID}
	assert.Equal(t, orgID, ftv.GetOrganizationID())
}

func TestFormulaTemplateVersion_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	ftv := &FormulaTemplateVersion{BusinessUnitID: buID}
	assert.Equal(t, buID, ftv.GetBusinessUnitID())
}

func TestFormulaTemplateVersion_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	ftv := &FormulaTemplateVersion{}
	config := ftv.GetPostgresSearchConfig()

	assert.Equal(t, "ftv", config.TableAlias)
	assert.True(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 4)
	assert.NotNil(t, config.Relationships)
	assert.Len(t, config.Relationships, 1)
	assert.Equal(t, "Template", config.Relationships[0].Field)
}

func TestNewVersionFromTemplate(t *testing.T) {
	t.Parallel()

	ft := &FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Test Template",
		Description:    "A test description",
		Type:           TemplateTypeFreightCharge,
		Expression:     "rate * weight",
		Status:         StatusActive,
		SchemaID:       "shipment",
	}

	createdByID := pulid.MustNew("usr_")
	changeMessage := "Initial version"
	changeSummary := map[string]jsonutils.FieldChange{
		"expression": {From: "rate", To: "rate * weight"},
	}

	ftv := NewVersionFromTemplate(ft, 1, createdByID, changeMessage, changeSummary)

	assert.Equal(t, ft.ID, ftv.TemplateID)
	assert.Equal(t, ft.OrganizationID, ftv.OrganizationID)
	assert.Equal(t, ft.BusinessUnitID, ftv.BusinessUnitID)
	assert.Equal(t, int64(1), ftv.VersionNumber)
	assert.Equal(t, ft.Name, ftv.Name)
	assert.Equal(t, ft.Description, ftv.Description)
	assert.Equal(t, ft.Type, ftv.Type)
	assert.Equal(t, ft.Expression, ftv.Expression)
	assert.Equal(t, ft.Status, ftv.Status)
	assert.Equal(t, ft.SchemaID, ftv.SchemaID)
	assert.Equal(t, createdByID, ftv.CreatedByID)
	assert.Equal(t, changeMessage, ftv.ChangeMessage)
	assert.Equal(t, changeSummary, ftv.ChangeSummary)
}

func TestNewVersionFromTemplate_WithMetadata(t *testing.T) {
	t.Parallel()

	ft := &FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Template with metadata",
		Expression:     "x + y",
		Type:           TemplateTypeAccessorialCharge,
		Status:         StatusDraft,
		SchemaID:       "shipment",
		Metadata:       map[string]any{"key": "value"},
	}

	ftv := NewVersionFromTemplate(ft, 5, pulid.MustNew("usr_"), "update", nil)

	assert.Equal(t, int64(5), ftv.VersionNumber)
	assert.Equal(t, ft.Metadata, ftv.Metadata)
	assert.Nil(t, ftv.ChangeSummary)
}

func TestVersionTag_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tag  VersionTag
		want bool
	}{
		{name: "stable is valid", tag: VersionTagStable, want: true},
		{name: "production is valid", tag: VersionTagProduction, want: true},
		{name: "draft is valid", tag: VersionTagDraft, want: true},
		{name: "testing is valid", tag: VersionTagTesting, want: true},
		{name: "deprecated is valid", tag: VersionTagDeprecated, want: true},
		{name: "empty is invalid", tag: VersionTag(""), want: false},
		{name: "unknown is invalid", tag: VersionTag("Unknown"), want: false},
		{name: "lowercase stable is invalid", tag: VersionTag("stable"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.tag.IsValid())
		})
	}
}

func TestVersionTag_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tag  VersionTag
		want string
	}{
		{name: "stable", tag: VersionTagStable, want: "Stable"},
		{name: "production", tag: VersionTagProduction, want: "Production"},
		{name: "draft", tag: VersionTagDraft, want: "Draft"},
		{name: "testing", tag: VersionTagTesting, want: "Testing"},
		{name: "deprecated", tag: VersionTagDeprecated, want: "Deprecated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.tag.String())
		})
	}
}

func TestStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{name: "active", status: StatusActive, want: "Active"},
		{name: "inactive", status: StatusInactive, want: "Inactive"},
		{name: "draft", status: StatusDraft, want: "Draft"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestStatusFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Status
		wantErr bool
	}{
		{name: "Active", input: "Active", want: StatusActive, wantErr: false},
		{name: "Inactive", input: "Inactive", want: StatusInactive, wantErr: false},
		{name: "Draft", input: "Draft", want: StatusDraft, wantErr: false},
		{name: "invalid", input: "Unknown", want: "", wantErr: true},
		{name: "empty", input: "", want: "", wantErr: true},
		{name: "lowercase active", input: "active", want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := StatusFromString(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTemplateType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tt   TemplateType
		want string
	}{
		{name: "freight charge", tt: TemplateTypeFreightCharge, want: "FreightCharge"},
		{name: "accessorial charge", tt: TemplateTypeAccessorialCharge, want: "AccessorialCharge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.tt.String())
		})
	}
}

func TestTemplateTypeFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    TemplateType
		wantErr bool
	}{
		{
			name:    "FreightCharge",
			input:   "FreightCharge",
			want:    TemplateTypeFreightCharge,
			wantErr: false,
		},
		{
			name:    "AccessorialCharge",
			input:   "AccessorialCharge",
			want:    TemplateTypeAccessorialCharge,
			wantErr: false,
		},
		{name: "invalid", input: "Unknown", want: "", wantErr: true},
		{name: "empty", input: "", want: "", wantErr: true},
		{name: "lowercase freight", input: "freightcharge", want: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := TemplateTypeFromString(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVersionDiff(t *testing.T) {
	t.Parallel()

	diff := VersionDiff{
		FromVersion: 1,
		ToVersion:   3,
		Changes: map[string]jsonutils.FieldChange{
			"expression": {From: "a + b", To: "a * b"},
			"name":       {From: "Old", To: "New"},
		},
		ChangeCount: 2,
	}

	assert.Equal(t, int64(1), diff.FromVersion)
	assert.Equal(t, int64(3), diff.ToVersion)
	assert.Equal(t, 2, diff.ChangeCount)
	assert.Len(t, diff.Changes, 2)
}

func TestForkLineage(t *testing.T) {
	t.Parallel()

	parentID := pulid.MustNew("ft_")
	sourceID := pulid.MustNew("ft_")
	sourceVersion := int64(2)

	lineage := ForkLineage{
		TemplateID:       parentID,
		TemplateName:     "Parent Template",
		SourceTemplateID: &sourceID,
		SourceVersion:    &sourceVersion,
		ForkedTemplates: []ForkLineage{
			{
				TemplateID:   pulid.MustNew("ft_"),
				TemplateName: "Child Template",
			},
		},
	}

	assert.Equal(t, parentID, lineage.TemplateID)
	assert.Equal(t, "Parent Template", lineage.TemplateName)
	assert.Equal(t, &sourceID, lineage.SourceTemplateID)
	assert.Equal(t, &sourceVersion, lineage.SourceVersion)
	assert.Len(t, lineage.ForkedTemplates, 1)
	assert.Equal(t, "Child Template", lineage.ForkedTemplates[0].TemplateName)
}

func TestForkLineage_NoSource(t *testing.T) {
	t.Parallel()

	lineage := ForkLineage{
		TemplateID:   pulid.MustNew("ft_"),
		TemplateName: "Root Template",
	}

	assert.Nil(t, lineage.SourceTemplateID)
	assert.Nil(t, lineage.SourceVersion)
	assert.Nil(t, lineage.ForkedTemplates)
}
