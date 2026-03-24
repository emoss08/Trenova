package dbhelper

import (
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
)

func TestWrapWildcard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"test", "%test%"},
		{"", "%%"},
		{"hello world", "%hello world%"},
		{"%already%", "%%already%%"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WrapWildcard(tt.input))
		})
	}
}

func TestSelectOptionsConfig_orgColumn(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{}
		assert.Equal(t, "organization_id", cfg.orgColumn())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{OrgColumn: "org_id"}
		assert.Equal(t, "org_id", cfg.orgColumn())
	})
}

func TestSelectOptionsConfig_buColumn(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{}
		assert.Equal(t, "business_unit_id", cfg.buColumn())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{BuColumn: "bu_id"}
		assert.Equal(t, "bu_id", cfg.buColumn())
	})
}

func TestSelectOptions_NilConfig(t *testing.T) {
	t.Parallel()
	req := &pagination.SelectQueryRequest{
		Pagination: pagination.Info{Limit: 10},
	}
	result, err := SelectOptions[any](t.Context(), nil, req, nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrSelectOptionsConfigRequired, err)
}

func TestSelectOptionsConfig_Columns(t *testing.T) {
	t.Parallel()

	t.Run("empty columns", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{}
		assert.Empty(t, cfg.Columns)
	})

	t.Run("with columns", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{
			Columns: []string{"id", "name", "code"},
		}
		assert.Equal(t, []string{"id", "name", "code"}, cfg.Columns)
	})
}

func TestSelectOptionsConfig_SearchColumns(t *testing.T) {
	t.Parallel()

	t.Run("empty search columns", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{}
		assert.Empty(t, cfg.SearchColumns)
	})

	t.Run("with search columns", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{
			SearchColumns: []string{"name", "description"},
		}
		assert.Equal(t, []string{"name", "description"}, cfg.SearchColumns)
	})
}

func TestSelectOptionsConfig_EntityName(t *testing.T) {
	t.Parallel()

	t.Run("empty entity name", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{}
		assert.Empty(t, cfg.EntityName)
	})

	t.Run("with entity name", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{EntityName: "fleet_codes"}
		assert.Equal(t, "fleet_codes", cfg.EntityName)
	})
}

func TestSelectOptionsConfig_QueryModifier(t *testing.T) {
	t.Parallel()

	t.Run("nil query modifier", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{}
		assert.Nil(t, cfg.QueryModifier)
	})

	t.Run("non-nil query modifier", func(t *testing.T) {
		t.Parallel()
		cfg := &SelectOptionsConfig{
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q
			},
		}
		assert.NotNil(t, cfg.QueryModifier)
	})
}

func TestSelectOptions_NilConfigWithQuery(t *testing.T) {
	t.Parallel()

	req := &pagination.SelectQueryRequest{
		Pagination: pagination.Info{Limit: 20, Offset: 0},
		TenantInfo: pagination.TenantInfo{},
		Query:      "search",
	}
	result, err := SelectOptions[any](t.Context(), nil, req, nil)
	assert.ErrorIs(t, err, ErrSelectOptionsConfigRequired)
	assert.Nil(t, result)
}

func TestSelectOptions_NilConfigWithTenantInfo(t *testing.T) {
	t.Parallel()

	req := &pagination.SelectQueryRequest{
		Pagination: pagination.Info{Limit: 50, Offset: 10},
		TenantInfo: pagination.TenantInfo{
			OrgID:  "org_123",
			BuID:   "bu_456",
			UserID: "usr_789",
		},
		Query: "",
	}
	result, err := SelectOptions[any](t.Context(), nil, req, nil)
	assert.ErrorIs(t, err, ErrSelectOptionsConfigRequired)
	assert.Nil(t, result)
}

func TestErrSelectOptionsConfigRequired(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "select options config is required", ErrSelectOptionsConfigRequired.Error())
}

func TestSelectOptionsConfig_FullConfig(t *testing.T) {
	t.Parallel()

	cfg := &SelectOptionsConfig{
		Columns:       []string{"id", "name", "code"},
		OrgColumn:     "org_id",
		BuColumn:      "bu_id",
		SearchColumns: []string{"name", "code"},
		EntityName:    "fleet_codes",
		QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
			return q
		},
	}

	assert.Equal(t, "org_id", cfg.orgColumn())
	assert.Equal(t, "bu_id", cfg.buColumn())
	assert.Equal(t, []string{"id", "name", "code"}, cfg.Columns)
	assert.Equal(t, []string{"name", "code"}, cfg.SearchColumns)
	assert.Equal(t, "fleet_codes", cfg.EntityName)
	assert.NotNil(t, cfg.QueryModifier)
}

func TestSelectOptionsConfig_DefaultColumns(t *testing.T) {
	t.Parallel()

	cfg := &SelectOptionsConfig{
		OrgColumn: "",
		BuColumn:  "",
	}

	assert.Equal(t, "organization_id", cfg.orgColumn())
	assert.Equal(t, "business_unit_id", cfg.buColumn())
}

func TestSelectOptionsConfig_WhitespaceColumns(t *testing.T) {
	t.Parallel()

	cfg := &SelectOptionsConfig{
		OrgColumn: "  custom_org  ",
		BuColumn:  "  custom_bu  ",
	}

	assert.Equal(t, "  custom_org  ", cfg.orgColumn())
	assert.Equal(t, "  custom_bu  ", cfg.buColumn())
}

func TestWrapWildcard_SpecialCharacters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"underscore", "test_value", "%test_value%"},
		{"percent", "50%off", "%50%off%"},
		{"single quote", "it's", "%it's%"},
		{"backslash", `test\value`, `%test\value%`},
		{"unicode", "日本語", "%日本語%"},
		{"spaces", "  spaces  ", "%  spaces  %"},
		{"newline", "line\nbreak", "%line\nbreak%"},
		{"tab", "tab\there", "%tab\there%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WrapWildcard(tt.input))
		})
	}
}

func TestSelectOptions_NilConfigZeroPagination(t *testing.T) {
	t.Parallel()

	req := &pagination.SelectQueryRequest{
		Pagination: pagination.Info{Limit: 0, Offset: 0},
	}
	result, err := SelectOptions[any](t.Context(), nil, req, nil)
	assert.ErrorIs(t, err, ErrSelectOptionsConfigRequired)
	assert.Nil(t, result)
}

func TestSelectOptionsConfig_MultipleSearchColumns(t *testing.T) {
	t.Parallel()

	cfg := &SelectOptionsConfig{
		SearchColumns: []string{"name", "code", "description", "notes"},
	}

	assert.Len(t, cfg.SearchColumns, 4)
	assert.Contains(t, cfg.SearchColumns, "name")
	assert.Contains(t, cfg.SearchColumns, "code")
	assert.Contains(t, cfg.SearchColumns, "description")
	assert.Contains(t, cfg.SearchColumns, "notes")
}

func TestSelectOptionsConfig_SingleColumn(t *testing.T) {
	t.Parallel()

	cfg := &SelectOptionsConfig{
		Columns: []string{"id"},
	}

	assert.Len(t, cfg.Columns, 1)
	assert.Equal(t, "id", cfg.Columns[0])
}
