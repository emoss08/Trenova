package compiler

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type consistencyRole struct {
	name       string
	engine     *fakePermissionEngine
	groups     permission.FieldGroupSet
	restricted bool
}

func consistencyRoles() []consistencyRole {
	public := allowAllEngine()
	public.defaultGrant.maxSensitivity = permission.SensitivityPublic

	shipmentsOnly := &fakePermissionEngine{
		defaultGrant: resourceGrant{maxSensitivity: permission.SensitivityRestricted},
		overrides: map[string]resourceGrant{
			"shipment": {
				operations:     []permission.Operation{permission.OpRead, permission.OpExport},
				dataScope:      permission.DataScopeOrganization,
				maxSensitivity: permission.SensitivityRestricted,
			},
			"shipment_move": {
				operations:     []permission.Operation{permission.OpRead},
				dataScope:      permission.DataScopeOrganization,
				maxSensitivity: permission.SensitivityRestricted,
			},
		},
	}

	return []consistencyRole{
		{name: "everything", engine: allowAllEngine(), groups: permission.AllFieldGroups()},
		{
			name:       "no customers, workers or locations, no protected groups",
			engine:     withoutGrant(allowAllEngine(), "customer", "worker", "location"),
			restricted: true,
		},
		{
			name:       "public ceiling",
			engine:     public,
			groups:     permission.AllFieldGroups(),
			restricted: true,
		},
		{
			name:       "shipments and moves only, personal fields",
			engine:     shipmentsOnly,
			groups:     permission.FieldGroupSet(0).With(permission.FieldGroupPersonal),
			restricted: true,
		},
	}
}

// consistencyField is one field a report can reach from a root: the field on
// the root itself, or on the target of one to-one edge out of it.
type consistencyField struct {
	root  string
	path  []string
	field string
}

func (f consistencyField) definition() *report.Definition {
	return &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    f.root,
		Columns:   []report.ColumnSpec{dim("c_probe", f.field, f.path...)},
	}
}

// TestCatalogAnnotationsAgreeWithTheCompiler walks the catalog the builder
// reads, for several roles, and compiles one report per field it marks: every
// available field compiles and every other field is refused. A field the
// compiler refuses for a reason other than authorization, such as a JSON
// column as a dimension, is not a field the builder offers as a column and is
// left out.
func TestCatalogAnnotationsAgreeWithTheCompiler(t *testing.T) {
	t.Parallel()

	catalog := &reportcatalog.Default
	unrestricted := newTestCompiler(allowAllEngine())

	for _, role := range consistencyRoles() {
		t.Run(role.name, func(t *testing.T) {
			t.Parallel()

			c := newTestCompilerWithGroups(role.engine, role.groups)
			registry := permission.NewRegistry()
			access, err := reportaccess.New(reportaccess.Params{
				Catalog:  catalog,
				Engine:   role.engine,
				Registry: registry,
				Views:    refview.NewPort(refview.New(registry), role.engine),
				Fields:   fakeFieldAccess{visible: role.groups},
			}).ForCatalog(t.Context(), newRequest(nil).Tenant)
			require.NoError(t, err)
			entities, err := access.Catalog(t.Context())
			require.NoError(t, err)

			decided := make(map[string]*reportaccess.Entity, len(entities))
			for i := range entities {
				decided[entities[i].Entity.Key] = &entities[i]
			}

			checked, refused := 0, 0
			check := func(probe consistencyField, available bool, why string) {
				def := probe.definition()
				if _, compileErr := unrestricted.ValidateAndAuthorize(
					t.Context(), newRequest(def),
				); compileErr != nil {
					return
				}
				checked++
				if !available {
					refused++
				}
				_, compileErr := c.ValidateAndAuthorize(t.Context(), newRequest(def))
				if available {
					assert.NoErrorf(t, compileErr, "%s is marked available and was refused", why)
					return
				}
				var authzErr *errortypes.AuthorizationError
				assert.ErrorAsf(t, compileErr, &authzErr,
					"%s is marked unavailable and was not refused", why)
			}

			for _, root := range entities {
				for i := range root.Entity.Fields {
					field := &root.Entity.Fields[i]
					check(
						consistencyField{root: root.Entity.Key, field: field.Key},
						root.Readable && root.Fields[i].Available(),
						root.Entity.Key+"."+field.Key,
					)
				}
				if !root.Readable {
					continue
				}

				for i := range root.Entity.Edges {
					edge := &root.Entity.Edges[i]
					target := decided[edge.Target]
					if !edge.Traversable || edge.Cardinality != reportcatalog.CardinalityOne ||
						target == nil {
						continue
					}
					for j := range target.Entity.Fields {
						field := &target.Entity.Fields[j]
						check(
							consistencyField{
								root:  root.Entity.Key,
								path:  []string{edge.Name},
								field: field.Key,
							},
							root.Edges[i].Available() && target.Fields[j].Available(),
							root.Entity.Key+"."+edge.Name+"."+field.Key,
						)
					}
				}
			}

			t.Logf("%d fields checked, %d of them marked unavailable", checked, refused)
			assert.Positive(t, checked)
			if role.restricted {
				assert.Positive(t, refused, "a restricted role is refused something")
			}
		})
	}
}
