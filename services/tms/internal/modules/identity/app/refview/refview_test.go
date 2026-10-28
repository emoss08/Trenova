package refview_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

type access struct {
	reads   map[permission.Resource]bool
	ceiling permission.FieldSensitivity
	applies bool
}

func (a access) Reads(_ context.Context, kind permission.Resource) bool { return a.reads[kind] }

func (a access) Ceiling(
	context.Context,
	permission.Resource,
	refview.Reach,
) (permission.FieldSensitivity, bool) {
	return a.ceiling, a.applies
}

func TestDecide_ANestedRecordShowsItsViewWithoutItsKindsRead(t *testing.T) {
	t.Parallel()

	views := refview.New(permissioncatalog.NewRegistry())
	none := access{}

	assert.Equal(t, refview.Visible,
		views.Decide(t.Context(), none, permission.ResourceWorker, "wholeName", refview.Nested))
	assert.Equal(t, refview.WithheldKind,
		views.Decide(t.Context(), none, permission.ResourceWorker, "phoneNumber", refview.Nested))
	assert.Equal(t, refview.Visible,
		views.Decide(t.Context(), none, permission.ResourceWorker, "id", refview.Nested))
}

func TestDecide_TheKindsReadShowsTheWholeRecord(t *testing.T) {
	t.Parallel()

	views := refview.New(permissioncatalog.NewRegistry())
	reader := access{reads: map[permission.Resource]bool{permission.ResourceWorker: true}}

	assert.Equal(t, refview.Visible,
		views.Decide(t.Context(), reader, permission.ResourceWorker, "phoneNumber", refview.Nested))
}

func TestDecide_RootAndSelfScopedRecordsAreNotGatedByKind(t *testing.T) {
	t.Parallel()

	views := refview.New(permissioncatalog.NewRegistry())
	for _, reach := range []refview.Reach{refview.Root, refview.SelfScoped} {
		assert.Equal(t, refview.Visible,
			views.Decide(t.Context(), access{}, permission.ResourceWorker, "phoneNumber", reach))
	}
}

func TestDecide_TheCeilingAppliesInsideAView(t *testing.T) {
	t.Parallel()

	views := refview.New(permissioncatalog.NewRegistry())
	public := access{ceiling: permission.SensitivityPublic, applies: true}

	assert.Equal(t, refview.WithheldSensitivity,
		views.Decide(t.Context(), public, permission.ResourceCustomer, "name", refview.Nested),
		"a view field is Internal, above a Public ceiling")
	assert.Equal(t, refview.Visible,
		views.Decide(t.Context(), public, permission.ResourceCustomer, "id", refview.Nested),
		"a record's id is never withheld")
}

func TestFilter_SplitsFieldsInOrder(t *testing.T) {
	t.Parallel()

	views := refview.New(permissioncatalog.NewRegistry())
	visible, withheld := views.Filter(t.Context(), access{}, permission.ResourceCustomer,
		[]string{"name", "creditLimit", "code"}, refview.Nested)

	assert.Equal(t, []string{"name", "code"}, visible)
	assert.Equal(t, []string{"creditLimit"}, withheld)
}

func TestPersonAccess_ReadsThroughTheEngineOncePerKind(t *testing.T) {
	t.Parallel()

	engine := &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorker.String()},
	}
	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	calls := 0
	person := refview.NewPersonAccess(engine, actor,
		func(ctx context.Context, kind permission.Resource) (bool, error) {
			calls++
			result, err := engine.Check(ctx, actor.PermissionCheck(kind, permission.OpRead))
			return err == nil && result.Allowed, err
		})

	assert.False(t, person.Reads(t.Context(), permission.ResourceWorker))
	assert.False(t, person.Reads(t.Context(), permission.ResourceWorker))
	assert.True(t, person.Reads(t.Context(), permission.ResourceTractor))
	assert.Equal(t, 2, calls)

	_, applies := person.Ceiling(t.Context(), permission.ResourceWorker, refview.Root)
	assert.False(t, applies, "a person's screen applies no ceiling to the records it asked for")
	_, applies = person.Ceiling(t.Context(), permission.ResourceWorker, refview.Nested)
	assert.True(t, applies)
}
