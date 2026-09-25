package fieldsensitivity

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

type fakeEngine struct {
	serviceports.PermissionEngine

	levels map[string]permission.FieldSensitivity
	fail   bool
	calls  atomic.Int32
}

func (f *fakeEngine) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*serviceports.ResourcePermissionDetail, error) {
	f.calls.Add(1)
	if f.fail {
		return nil, errors.New("engine unavailable")
	}

	return &serviceports.ResourcePermissionDetail{
		Resource:       resource,
		MaxSensitivity: f.levels[resource],
	}, nil
}

func TestPersonCeiling_NeverRisesAboveRestricted(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{levels: map[string]permission.FieldSensitivity{
		permission.ResourceWorker.String():  permission.SensitivityConfidential,
		permission.ResourceInvoice.String(): permission.SensitivityInternal,
	}}
	user, org := pulid.MustNew("usr_"), pulid.MustNew("org_")

	assert.Equal(t, permission.SensitivityRestricted,
		PersonCeiling(t.Context(), engine, user, org, permission.ResourceWorker))
	assert.Equal(t, permission.SensitivityInternal,
		PersonCeiling(t.Context(), engine, user, org, permission.ResourceInvoice))
}

func TestPersonCeiling_AnUnresolvedAuthorizationReadsAtInternal(t *testing.T) {
	t.Parallel()

	user, org := pulid.MustNew("usr_"), pulid.MustNew("org_")

	assert.Equal(t, permission.SensitivityInternal,
		PersonCeiling(t.Context(), &fakeEngine{fail: true}, user, org, permission.ResourceWorker))
	assert.Equal(t, permission.SensitivityInternal,
		PersonCeiling(t.Context(), nil, user, org, permission.ResourceWorker))
}

func TestVisible_ConfidentialIsNeverShown(t *testing.T) {
	t.Parallel()

	assert.False(
		t,
		VisibleAt(permission.SensitivityConfidential, permission.SensitivityConfidential),
	)
	assert.True(t, VisibleAt(permission.SensitivityRestricted, permission.SensitivityRestricted))
	assert.False(t, VisibleAt(permission.SensitivityRestricted, permission.SensitivityInternal))
	assert.True(t, VisibleAt(permission.SensitivityInternal, permission.SensitivityInternal))
}

func TestLevel_UnknownResourceIsInternal(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()

	assert.Equal(t, permission.SensitivityInternal,
		Level(registry, permission.Resource("no_such_resource"), "anything"))
	assert.Equal(t, permission.SensitivityInternal,
		Level(nil, permission.ResourceWorker, "anything"))
}

func TestCeilings_AskTheEngineOncePerResource(t *testing.T) {
	t.Parallel()

	engine := &fakeEngine{levels: map[string]permission.FieldSensitivity{
		permission.ResourceWorker.String(): permission.SensitivityRestricted,
	}}
	ceilings := NewCeilings(engine, pulid.MustNew("usr_"), pulid.MustNew("org_"))

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			assert.Equal(t, permission.SensitivityRestricted,
				ceilings.For(t.Context(), permission.ResourceWorker))
		})
	}
	wg.Wait()

	assert.Equal(t, permission.SensitivityRestricted,
		ceilings.For(t.Context(), permission.ResourceWorker))
	assert.LessOrEqual(t, engine.calls.Load(), int32(8))
	before := engine.calls.Load()
	ceilings.For(t.Context(), permission.ResourceWorker)
	assert.Equal(t, before, engine.calls.Load(), "a remembered ceiling is not asked again")
}

type fakeChecker struct {
	allowed map[string]bool
	fail    bool
	calls   atomic.Int32
}

func (f *fakeChecker) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	f.calls.Add(1)
	if f.fail {
		return nil, errors.New("engine unavailable")
	}

	return &serviceports.PermissionCheckResult{Allowed: f.allowed[req.Resource]}, nil
}

func TestReadAccess_AsksOncePerResourceAndRefusesWhatItCannotCheck(t *testing.T) {
	t.Parallel()

	actor := &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	checker := &fakeChecker{allowed: map[string]bool{permission.ResourceShipment.String(): true}}
	access := NewReadAccess(checker, actor)

	assert.True(t, access.MayRead(t.Context(), permission.ResourceShipment))
	assert.True(t, access.MayRead(t.Context(), permission.ResourceShipment))
	assert.False(t, access.MayRead(t.Context(), permission.ResourceInvoice))
	assert.Equal(t, int32(2), checker.calls.Load())

	failing := NewReadAccess(&fakeChecker{fail: true}, actor)
	assert.False(t, failing.MayRead(t.Context(), permission.ResourceShipment))
	assert.False(t, NewReadAccess(checker, nil).MayRead(t.Context(), permission.ResourceShipment))

	var none *ReadAccess
	assert.False(t, none.MayRead(t.Context(), permission.ResourceShipment))
}
