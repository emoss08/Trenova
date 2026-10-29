package compliancejobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const sweepNow = int64(1_767_225_600)

type recordingPublisher struct {
	events []agenteventports.AgentEvent
}

func (r *recordingPublisher) Publish(_ context.Context, event *agenteventports.AgentEvent) {
	r.events = append(r.events, *event)
}

type recordingProjector struct {
	items []watchtowerports.WatchtowerItemInput
}

func (r *recordingProjector) Upsert(_ context.Context, item watchtowerports.WatchtowerItemInput) {
	r.items = append(r.items, item)
}

func (r *recordingProjector) Resolve(
	context.Context,
	pagination.TenantInfo,
	watchtower.SourceKind,
	string,
) {
}

func credentialOf(
	workerID pulid.ID,
	name string,
	daysLeft int64,
) *workercredential.WorkerCredential {
	expires := sweepNow + daysLeft*secondsPerDay

	return &workercredential.WorkerCredential{
		ID:        pulid.MustNew("wcred_"),
		WorkerID:  workerID,
		ExpiresAt: &expires,
		Worker: &worker.Worker{
			ID:        workerID,
			FirstName: "Dana",
			LastName:  "Reyes",
		},
		CredentialType: &workercredential.WorkerCredentialType{Name: name},
	}
}

func sweptState() *sweepState {
	return &sweepState{
		now:            sweepNow,
		remindersByOrg: make(map[pulid.ID]bool),
		touchedWorkers: make(map[pulid.ID]pagination.TenantInfo),
		dueDrivers:     make(map[pulid.ID]*dueDriver),
		result:         new(CredentialExpirySweepResult),
	}
}

// A driver with three papers coming due is one renewal packet, so the sweep
// raises the driver once however many credentials reached a mark. The
// notification path deduplicates per credential and would fire three times.
func TestRaiseDueDrivers_OncePerDriverHoweverManyPapers(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	workerID := pulid.MustNew("wrk_")

	state := sweptState()
	for _, cred := range []*workercredential.WorkerCredential{
		credentialOf(workerID, "CDL", 30),
		credentialOf(workerID, "Medical card", 14),
		credentialOf(workerID, "Hazmat endorsement", 7),
	} {
		state.recordDueDriver(
			cred,
			tenantInfo,
			workercredential.DaysUntil(*cred.ExpiresAt, sweepNow),
		)
	}

	publisher := &recordingPublisher{}
	projector := &recordingProjector{}
	activities := &Activities{
		logger:     zap.NewNop(),
		publisher:  publisher,
		watchtower: projector,
	}

	activities.raiseDueDrivers(t.Context(), state)

	require.Len(t, publisher.events, 1)
	require.Equal(t, agenttrigger.EventWorkerCredentialExpiring, publisher.events[0].Kind)
	require.Equal(t, workerID, publisher.events[0].SubjectID)
	require.Equal(t, tenantInfo, publisher.events[0].TenantInfo)
	require.Equal(t, 1, state.result.DriversRaised)

	require.Len(t, projector.items, 1)
	item := projector.items[0]
	require.Equal(t, watchtower.SourceWorkerCredential, item.SourceKind)
	require.Equal(t, workerID.String(), item.SourceID)
	require.Equal(t, agent.SubjectWorker, item.SubjectType)
	require.Contains(t, item.Title, "Dana Reyes")
	require.Contains(t, item.Title, "3 credentials")
	require.Contains(t, item.Summary, "CDL")
	require.Contains(t, item.Summary, "Medical card")
	require.Contains(t, item.Summary, "Hazmat endorsement")
	// The soonest paper decides the severity, and seven days is a problem
	// rather than a reminder.
	require.Equal(t, watchtower.SeverityWarning, item.Severity)
}

func TestRaiseDueDrivers_SeparateDriversAreSeparateSubjects(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	first, second := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")

	state := sweptState()
	state.recordDueDriver(credentialOf(first, "CDL", 30), tenantInfo, 30)
	state.recordDueDriver(credentialOf(second, "CDL", 30), tenantInfo, 30)

	publisher := &recordingPublisher{}
	activities := &Activities{logger: zap.NewNop(), publisher: publisher}
	activities.raiseDueDrivers(t.Context(), state)

	require.Len(t, publisher.events, 2)
	require.ElementsMatch(t,
		[]pulid.ID{first, second},
		[]pulid.ID{publisher.events[0].SubjectID, publisher.events[1].SubjectID},
	)
}

func TestRaiseDueDrivers_AnExpiredPaperIsCritical(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	workerID := pulid.MustNew("wrk_")

	state := sweptState()
	state.recordDueDriver(credentialOf(workerID, "CDL", 30), tenantInfo, 30)
	state.recordDueDriver(credentialOf(workerID, "Medical card", -2), tenantInfo, -2)

	projector := &recordingProjector{}
	activities := &Activities{
		logger:     zap.NewNop(),
		publisher:  &recordingPublisher{},
		watchtower: projector,
	}
	activities.raiseDueDrivers(t.Context(), state)

	require.Len(t, projector.items, 1)
	require.Equal(t, watchtower.SeverityCritical, projector.items[0].Severity)
	require.Contains(t, projector.items[0].Summary, "expired 2d ago")
}
