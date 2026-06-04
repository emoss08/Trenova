package projection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelect_MapsScalarSelectionToColumns(t *testing.T) {
	t.Parallel()

	spec := TypeSpec{
		TypeName: "Test",
		FieldMap: map[string]string{
			"id":   "id",
			"code": "code",
		},
		AlwaysColumns: []string{"id"},
		Fields: []FieldSpec{
			{Name: "id", FieldMapKey: "id"},
			{Name: "code", FieldMapKey: "code"},
		},
	}

	selection := Select(spec, func(path string) bool {
		return path == "code"
	}, SelectOptions{})

	assert.Equal(t, []string{"id", "code"}, selection.Columns)
}

func TestSelect_DedupesAliases(t *testing.T) {
	t.Parallel()

	spec := TypeSpec{
		TypeName: "Test",
		FieldMap: map[string]string{
			"id":     "id",
			"status": "status",
		},
		AlwaysColumns: []string{"id"},
		Fields: []FieldSpec{
			{Name: "status", FieldMapKey: "status"},
			{Name: "equipmentStatus", FieldMapKey: "status"},
		},
	}

	selection := Select(spec, func(path string) bool {
		return path == "status" || path == "equipmentStatus"
	}, SelectOptions{})

	assert.Equal(t, []string{"id", "status"}, selection.Columns)
}

func TestSelect_SelectsRelationColumns(t *testing.T) {
	t.Parallel()

	child := TypeSpec{
		TypeName:      "Child",
		FieldMap:      map[string]string{"id": "id", "name": "name"},
		AlwaysColumns: []string{"id"},
		Fields: []FieldSpec{
			{Name: "id", FieldMapKey: "id"},
			{Name: "name", FieldMapKey: "name"},
		},
	}
	parent := TypeSpec{
		TypeName:      "Parent",
		FieldMap:      map[string]string{"id": "id"},
		AlwaysColumns: []string{"id"},
		Fields: []FieldSpec{
			{
				Name: "child",
				Relation: &RelationSpec{
					Target: &child,
				},
			},
		},
	}

	selection := Select(parent, func(path string) bool {
		return path == "child" || path == "child.name"
	}, SelectOptions{})

	assert.True(t, selection.HasRelation("child"))
	assert.Equal(t, []string{"id", "name"}, selection.RelationColumns("child"))
}

func TestSelect_GatesSuppressRelations(t *testing.T) {
	t.Parallel()

	child := TypeSpec{
		TypeName:      "Child",
		FieldMap:      map[string]string{"id": "id"},
		AlwaysColumns: []string{"id"},
	}
	parent := TypeSpec{
		TypeName:      "Parent",
		FieldMap:      map[string]string{"id": "id"},
		AlwaysColumns: []string{"id"},
		Fields: []FieldSpec{
			{
				Name: "child",
				Relation: &RelationSpec{
					Target: &child,
					Gate:   "details",
				},
			},
		},
	}

	selection := Select(parent, func(path string) bool {
		return path == "child"
	}, SelectOptions{
		Gates: map[string]bool{"details": false},
	})

	assert.False(t, selection.HasRelation("child"))
	assert.Equal(t, []string{"id"}, selection.Columns)
}

func TestSelect_VirtualFieldsSetSpecials(t *testing.T) {
	t.Parallel()

	spec := TypeSpec{
		TypeName:      "Test",
		FieldMap:      map[string]string{"id": "id"},
		AlwaysColumns: []string{"id"},
		Fields: []FieldSpec{
			{Name: "customFields", Special: "customFields"},
		},
	}

	selection := Select(spec, func(path string) bool {
		return path == "customFields"
	}, SelectOptions{})

	assert.Equal(t, []string{"id"}, selection.Columns)
	assert.True(t, selection.HasSpecial("customFields"))
}
