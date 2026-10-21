package reportcatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryFieldAndEdgeIsClassifiedByItsReferenceView(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	for i := range Default.Entities {
		entity := &Default.Entities[i]
		for j := range entity.Fields {
			field := &entity.Fields[j]
			assertClass(t, registry, entity, "field", field.Key, field.Class)
		}
		for j := range entity.Edges {
			edge := &entity.Edges[j]
			assertClass(t, registry, entity, "edge", edge.Name, edge.Class)
		}
	}
}

func assertClass(
	t *testing.T,
	registry *permission.Registry,
	entity *Entity,
	what, key string,
	class FieldClass,
) {
	t.Helper()

	if !assert.Truef(t, class.IsValid(),
		"%s %s.%s has no class; run `task generate-reportcatalog` so the generator classifies it",
		what, entity.Key, key) {
		return
	}

	want := Classify(registry, entity, key)
	if want == class {
		return
	}
	switch {
	case entity.WithinRecord:
		assert.Failf(t, "classified as part of a record's view",
			"%s %s.%s is %q, but %s is withinRecord, so none of its fields is in the %s "+
				"reference view; run `task generate-reportcatalog`",
			what, entity.Key, key, class, entity.Key, entity.Resource)
	case class == ClassReference:
		assert.Failf(t, "reference field outside its view",
			"%s %s.%s is classified reference, but %q is not in the %s ReferenceView "+
				"(resources_*.go); run `task generate-reportcatalog`, or, if linking reports "+
				"genuinely need it, add it to the view with a security review "+
				"(docs/engineering/reference-views.md)",
			what, entity.Key, key, key, entity.Resource)
	default:
		assert.Failf(t, "view field classified kind_read",
			"%s %s.%s is classified kind_read, but %q is in the %s ReferenceView; run "+
				"`task generate-reportcatalog`, or mark %s withinRecord: true in "+
				"reportcatalog.yml if its rows are only part of a %s record",
			what, entity.Key, key, key, entity.Resource, entity.Key, entity.Resource)
	}
}

func TestOnlyOneEntityIsTheRecordOfAResource(t *testing.T) {
	t.Parallel()

	records := make(map[permission.Resource]string, len(Default.Entities))
	for i := range Default.Entities {
		entity := &Default.Entities[i]
		if entity.WithinRecord {
			continue
		}
		other, taken := records[entity.Resource]
		assert.Falsef(t, taken,
			"%s and %s both show the %s reference view; mark the one that only holds rows of "+
				"a %s record withinRecord: true in reportcatalog.yml",
			other, entity.Key, entity.Resource, entity.Resource)
		records[entity.Resource] = entity.Key
	}
}

func TestClassificationFollowsTheRecordNotTheResource(t *testing.T) {
	t.Parallel()

	cases := []struct {
		entity string
		key    string
		want   FieldClass
	}{
		{"shipment", "proNumber", ClassReference},
		{"shipment", "totalChargeAmount", ClassKindRead},
		{"customer", "name", ClassReference},
		{"customer", "dotNumber", ClassKindRead},
		{"worker", "firstName", ClassReference},
		{"worker", "phoneNumber", ClassKindRead},
		{"worker_profile", "id", ClassKindRead},
		{"worker_profile", "licenseNumber", ClassKindRead},
		{"invoice_line", "id", ClassKindRead},
		{"assignment", "id", ClassKindRead},
	}
	for _, tc := range cases {
		entity, ok := Default.Entity(tc.entity)
		require.Truef(t, ok, "entity %s", tc.entity)
		got := classOf(entity, tc.key)
		assert.Equalf(t, tc.want, got, "%s.%s", tc.entity, tc.key)
	}
}

func TestEveryWorkerProfileFieldNeedsWorkerRead(t *testing.T) {
	t.Parallel()

	profile, ok := Default.Entity("worker_profile")
	require.True(t, ok)
	require.Equal(t, permission.ResourceWorker, profile.Resource)
	for i := range profile.Fields {
		assert.Equalf(t, ClassKindRead, profile.Fields[i].Class,
			"worker_profile.%s is not in the worker reference view", profile.Fields[i].Key)
	}
}

func classOf(entity *Entity, key string) FieldClass {
	if field, ok := entity.Field(key); ok {
		return field.Class
	}
	if edge, ok := entity.Edge(key); ok {
		return edge.Class
	}
	return ""
}
