package compliancejobs

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

const (
	eventDriverDigest = "dash.obligations_digest"
	// A digest looks a little further ahead than it fires, so a driver sees a
	// renewal before the week it is due rather than on the morning of it.
	dailyDigestHorizonDays  = 7
	weeklyDigestHorizonDays = 14
	// digestMaxItems caps a notice at a length somebody will actually read.
	// Anything beyond it is counted rather than listed.
	digestMaxItems = 12
)

// DriverDigestSweepResult reports what the digest pass sent.
type DriverDigestSweepResult struct {
	OrganizationsBundling int `json:"organizationsBundling"`
	DriversNotified       int `json:"driversNotified"`
	Obligations           int `json:"obligations"`
	Skipped               int `json:"skipped"`
	Failed                int `json:"failed"`
}

type digestState struct {
	now int64
	// controls caches each organisation's Dash settings. The sweep walks rows
	// across every tenant, so without it the same control would be read once
	// per credential.
	controls map[pagination.TenantInfo]*tenant.DashControl
	// items collects one bucket per driver. The obligations are gathered first
	// and sent once, which is the whole point of a digest.
	items  map[pulid.ID]*digestBucket
	result *DriverDigestSweepResult
}

type digestBucket struct {
	tenantInfo pagination.TenantInfo
	cadence    tenant.DriverDigestCadence
	items      []documenttemplate.DriverObligation
}

// DriverDigestActivity bundles everything a driver owes into one notice, for
// the organisations that asked for a daily or weekly cadence.
//
// It walks the same lists the reminder sweeps do rather than sharing state
// with them: activities are separate invocations and may not even run on the
// same worker, so a bucket filled in one and read in another would be empty
// often enough to be a bug nobody could reproduce. The reminder sweeps
// suppress their own per-item driver notices while a bundling cadence is in
// force, so a driver is told once rather than twice.
func (a *Activities) DriverDigestActivity(
	ctx context.Context,
) (*DriverDigestSweepResult, error) {
	state := &digestState{
		now:      timeutils.NowUnix(),
		controls: make(map[pagination.TenantInfo]*tenant.DashControl),
		items:    make(map[pulid.ID]*digestBucket),
		result:   new(DriverDigestSweepResult),
	}

	if err := a.collectCredentialObligations(ctx, state); err != nil {
		return nil, err
	}
	if err := a.collectTrainingObligations(ctx, state); err != nil {
		return nil, err
	}

	a.flushDigests(ctx, state)

	return state.result, nil
}

// digestControl reads and caches an organisation's Dash settings, and reports
// whether a digest should go out for it tonight.
func (a *Activities) digestControl(
	ctx context.Context,
	state *digestState,
	tenantInfo pagination.TenantInfo,
) (*tenant.DashControl, bool, error) {
	control, ok := state.controls[tenantInfo]
	if !ok {
		loaded, err := a.dashControlRepo.GetOrCreate(ctx, tenantInfo)
		if err != nil {
			return nil, false, err
		}
		control = loaded
		state.controls[tenantInfo] = control
		if control.DriverDigestCadence.Bundles() && control.SendCredentialReminders {
			state.result.OrganizationsBundling++
		}
	}
	return control, digestDueToday(control, state.now), nil
}

// digestDueToday says whether tonight is a night this organisation's digest
// goes out. A weekly digest that fired every night would not be a weekly one.
func digestDueToday(control *tenant.DashControl, now int64) bool {
	if control == nil || !control.SendCredentialReminders {
		return false
	}
	switch control.DriverDigestCadence {
	case tenant.DigestDaily:
		return true
	case tenant.DigestWeekly:
		return int16(time.Unix(now, 0).UTC().Weekday()) == control.DriverDigestWeekday
	case tenant.DigestImmediate:
		return false
	default:
		return false
	}
}

// digestHorizon is how far ahead a cadence looks. A weekly notice covers the
// fortnight so nothing falls due in the gap between two of them.
func digestHorizon(cadence tenant.DriverDigestCadence) int64 {
	if cadence == tenant.DigestWeekly {
		return weeklyDigestHorizonDays
	}
	return dailyDigestHorizonDays
}

func (a *Activities) collectCredentialObligations(
	ctx context.Context,
	state *digestState,
) error {
	var after pulid.ID
	for {
		page, err := a.credentialRepo.ListExpiring(
			ctx,
			&repositories.ListExpiringWorkerCredentialsRequest{
				HorizonDays: sweepHorizonDays,
				GraceDays:   sweepGraceDays,
				AfterID:     after,
				Limit:       sweepPageSize,
			},
		)
		if err != nil {
			return err
		}
		for _, cred := range page {
			if cred.Worker == nil || cred.CredentialType == nil || cred.ExpiresAt == nil {
				continue
			}
			tenantInfo := pagination.TenantInfo{
				OrgID: cred.OrganizationID,
				BuID:  cred.BusinessUnitID,
			}
			control, due, cErr := a.digestControl(ctx, state, tenantInfo)
			if cErr != nil {
				return cErr
			}
			if !due || cred.Worker.UserID.IsNil() {
				continue
			}
			daysLeft := worker.DaysUntil(*cred.ExpiresAt, state.now)
			if daysLeft > digestHorizon(control.DriverDigestCadence) {
				continue
			}
			state.add(tenantInfo, control, cred.WorkerID, documenttemplate.DriverObligation{
				What:      cred.CredentialType.Name,
				Kind:      "credential",
				DueAt:     credentialExpiryDate(cred),
				DueInDays: int(daysLeft),
				Overdue:   daysLeft < 0,
			})
		}
		activity.RecordHeartbeat(ctx, state.result.Obligations)
		if len(page) < sweepPageSize {
			return nil
		}
		after = page[len(page)-1].ID
	}
}

func (a *Activities) collectTrainingObligations(ctx context.Context, state *digestState) error {
	lists := []trainingLister{a.trainingRepo.ListDue, a.trainingRepo.ListExpiring}
	for _, list := range lists {
		if err := a.collectTrainingList(ctx, state, list); err != nil {
			return err
		}
	}
	return nil
}

func (a *Activities) collectTrainingList(
	ctx context.Context,
	state *digestState,
	list trainingLister,
) error {
	var after pulid.ID
	for {
		page, err := list(ctx, &repositories.ListTrainingRemindersRequest{
			HorizonDays: sweepHorizonDays,
			GraceDays:   sweepGraceDays,
			AfterID:     after,
			Limit:       sweepPageSize,
		})
		if err != nil {
			return err
		}
		for _, record := range page {
			if record.Worker == nil || record.Course == nil {
				continue
			}
			// A record is due, or lapsing, or neither; the expiry is what a
			// renewal hangs on and the due date what a new course does.
			at := record.DueAt
			if at == nil {
				at = record.ExpiresAt
			}
			if at == nil {
				continue
			}
			tenantInfo := pagination.TenantInfo{
				OrgID: record.OrganizationID,
				BuID:  record.BusinessUnitID,
			}
			control, due, cErr := a.digestControl(ctx, state, tenantInfo)
			if cErr != nil {
				return cErr
			}
			if !due || record.Worker.UserID.IsNil() {
				continue
			}
			daysLeft := worker.DaysUntil(*at, state.now)
			if daysLeft > digestHorizon(control.DriverDigestCadence) {
				continue
			}
			state.add(tenantInfo, control, record.WorkerID, documenttemplate.DriverObligation{
				What:      record.Course.Name,
				Kind:      "training",
				DueAt:     trainingDate(at),
				DueInDays: int(daysLeft),
				Overdue:   daysLeft < 0,
			})
		}
		activity.RecordHeartbeat(ctx, state.result.Obligations)
		if len(page) < sweepPageSize {
			return nil
		}
		after = page[len(page)-1].ID
	}
}

func (s *digestState) add(
	tenantInfo pagination.TenantInfo,
	control *tenant.DashControl,
	workerID pulid.ID,
	item documenttemplate.DriverObligation,
) {
	bucket, ok := s.items[workerID]
	if !ok {
		bucket = &digestBucket{
			tenantInfo: tenantInfo,
			cadence:    control.DriverDigestCadence,
		}
		s.items[workerID] = bucket
	}
	bucket.items = append(bucket.items, item)
	s.result.Obligations++
}

// flushDigests sends one notice per driver. Best-effort per driver: one
// driver's notice failing must not abandon the rest of the round.
func (a *Activities) flushDigests(ctx context.Context, state *digestState) {
	for workerID, bucket := range state.items {
		if len(bucket.items) == 0 {
			continue
		}
		sent, err := a.sendDigest(ctx, state, workerID, bucket)
		switch {
		case err != nil:
			state.result.Failed++
			a.logger.Warn("failed to send driver digest",
				zap.String("workerId", workerID.String()),
				zap.Error(err))
		case sent:
			state.result.DriversNotified++
		default:
			state.result.Skipped++
		}
		activity.RecordHeartbeat(ctx, state.result.DriversNotified)
	}
}

func (a *Activities) sendDigest(
	ctx context.Context,
	state *digestState,
	workerID pulid.ID,
	bucket *digestBucket,
) (bool, error) {
	// The period key is what makes the send idempotent: a retried activity
	// resolves to the same key and the notification is recognised as already
	// sent, while tomorrow's run resolves to a new one.
	correlation := fmt.Sprintf(
		"digest-%s-%s",
		workerID,
		digestPeriodKey(bucket.cadence, state.now),
	)
	exists, err := a.alreadySent(ctx, bucket.tenantInfo, eventDriverDigest, correlation)
	if err != nil || exists {
		return false, err
	}

	items := bucket.items
	// Overdue first, then soonest: a driver reads the top of a list.
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].DueInDays < items[j].DueInDays
	})
	if len(items) > digestMaxItems {
		items = items[:digestMaxItems]
	}

	a.driverNotify.NotifyWithCorrelation(ctx, &drivernotificationservice.DriverNotification{
		TenantInfo: bucket.tenantInfo,
		WorkerID:   workerID,
		EventType:  eventDriverDigest,
		Priority:   digestPriority(items),
		Context: documenttemplate.DriverNotificationContext{
			Digest:       items,
			DigestPeriod: digestPeriodLabel(bucket.cadence),
		},
		Link: "/dash/profile",
	}, correlation)
	return true, nil
}

// digestPriority lifts the notice when something has already lapsed. A
// round-up of things due next week is not the same message as one saying a
// licence expired on Tuesday.
func digestPriority(items []documenttemplate.DriverObligation) notification.Priority {
	for _, item := range items {
		if item.Overdue {
			return notification.PriorityHigh
		}
	}
	return notification.PriorityMedium
}

func digestPeriodKey(cadence tenant.DriverDigestCadence, now int64) string {
	at := time.Unix(now, 0).UTC()
	if cadence == tenant.DigestWeekly {
		year, week := at.ISOWeek()
		return fmt.Sprintf("%dW%02d", year, week)
	}
	return at.Format("2006-01-02")
}

func digestPeriodLabel(cadence tenant.DriverDigestCadence) string {
	if cadence == tenant.DigestWeekly {
		return "this week"
	}
	return "today"
}
