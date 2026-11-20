package storagequotaservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingNotifier struct {
	warned   []notificationservice.NotifyPermittedRequest
	resolved []notificationports.ResolveNotificationSubjectRequest
}

func (r *recordingNotifier) NotifyPermitted(
	_ context.Context,
	req notificationservice.NotifyPermittedRequest,
) (int, error) {
	r.warned = append(r.warned, req)
	return 1, nil
}

func (r *recordingNotifier) ResolveSubject(
	_ context.Context,
	req notificationports.ResolveNotificationSubjectRequest,
) (int64, error) {
	r.resolved = append(r.resolved, req)
	return 1, nil
}

type twoTenants struct{}

func (twoTenants) ListTenantsPage(
	_ context.Context,
	after pulid.ID,
	_ int,
) ([]pagination.TenantInfo, error) {
	if after.IsNotNil() {
		return nil, nil
	}
	return []pagination.TenantInfo{
		{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}, nil
}

func storageAt(used int64) *managedusage.State {
	return &managedusage.State{
		WindowStart: 1,
		WindowEnd:   2,
		Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterDocumentStorageBytes,
			Unit:      managedusage.UnitBytes,
			Quantity:  used,
			Allowance: 1000,
		}},
	}
}

func TestTheStorageNoticeFiresOncePerCrossing(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	w := &Warnings{
		notifier: notifier,
		tenants:  twoTenants{},
		clock:    func() int64 { return 100 },
		l:        zap.NewNop(),
	}
	steps := []struct {
		previous *managedusage.State
		current  *managedusage.State
		warned   int
		resolved int
	}{
		{previous: nil, current: storageAt(500), warned: 0, resolved: 0},
		{previous: storageAt(500), current: storageAt(850), warned: 2, resolved: 0},
		{previous: storageAt(850), current: storageAt(900), warned: 2, resolved: 0},
		{previous: storageAt(900), current: storageAt(700), warned: 2, resolved: 2},
		{previous: storageAt(700), current: storageAt(810), warned: 4, resolved: 2},
	}
	for _, step := range steps {
		require.NoError(t, w.StateChanged(t.Context(), &managedusageports.StateChange{
			Scope:    managedscope.Instance(),
			Previous: step.previous,
			Current:  step.current,
		}))
		assert.Len(t, notifier.warned, step.warned)
		assert.Len(t, notifier.resolved, step.resolved)
	}

	notice := notifier.warned[0]
	assert.Equal(t, EventStorageNearlyFull, notice.Notification.EventType)
	assert.Equal(t, notifier.resolved[0].SubjectKey, notice.Notification.SubjectKey)
	assert.Contains(t, notice.Notification.Message, "of the")
}

func TestTheFirstStateAlreadyPastTheShareWarns(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	w := &Warnings{
		notifier: notifier,
		tenants:  twoTenants{},
		clock:    func() int64 { return 100 },
		l:        zap.NewNop(),
	}

	require.NoError(t, w.StateChanged(t.Context(), &managedusageports.StateChange{
		Scope:   managedscope.Instance(),
		Current: storageAt(950),
	}))
	assert.Len(t, notifier.warned, 2, "an instance's state concerns every organization on it")
}

func TestAnOrganizationsStorageNoticeGoesToItsOwnAdministratorsOnly(t *testing.T) {
	t.Parallel()

	notifier := &recordingNotifier{}
	w := &Warnings{
		notifier: notifier,
		tenants:  twoTenants{},
		clock:    func() int64 { return 100 },
		l:        zap.NewNop(),
	}
	tenant := dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	require.NoError(t, w.StateChanged(t.Context(), &managedusageports.StateChange{
		Scope:   managedscope.Organization(tenant),
		Current: storageAt(950),
	}))
	require.Len(t, notifier.warned, 1)
	assert.Equal(t, tenant.OrganizationID, notifier.warned[0].Tenant.OrgID)
	assert.Equal(t, tenant.BusinessUnitID, notifier.warned[0].Tenant.BuID)

	require.NoError(t, w.StateChanged(t.Context(), &managedusageports.StateChange{
		Scope:    managedscope.Organization(tenant),
		Previous: storageAt(950),
		Current:  storageAt(100),
	}))
	require.Len(t, notifier.resolved, 1)
	assert.Equal(t, tenant.OrganizationID, notifier.resolved[0].OrganizationID)
}
