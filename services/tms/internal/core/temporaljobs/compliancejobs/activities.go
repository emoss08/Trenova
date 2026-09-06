package compliancejobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/internal/core/services/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/core/services/workersafetyservice"
	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	sweepHorizonDays = 30
	sweepGraceDays   = 7
	sweepPageSize    = 500
	dedupeWindowDays = 6
	secondsPerDay    = int64(86400)

	eventDriverExpiring     = "dash.credential_expiring"
	eventComplianceExpiring = "credential_expiring"
	eventComplianceExpired  = "credential_expired"
	complianceLink          = "/hr/workers?tab=credentials"
)

// reminderSteps are the days-until-expiry marks at which a driver and the
// back office are reminded; day 0 and every day past expiry always fire, the
// dedupe window keeps that to one notice a week.
var reminderSteps = []int64{30, 14, 7}

type ActivitiesParams struct {
	fx.In

	CredentialRepo  repositories.WorkerCredentialRepository
	Credentials     *workercredentialservice.Service
	TrainingRepo    repositories.WorkerTrainingRepository
	Training        *workertrainingservice.Service
	SafetyRepo      repositories.WorkerSafetyRepository
	Safety          *workersafetyservice.Service
	DrugAlcoholRepo repositories.WorkerDrugAlcoholRepository
	DrugAlcohol     *workerdrugalcoholservice.Service
	DashControlRepo repositories.DashControlRepository
	Notifications   *notificationservice.Service
	DriverNotify    *drivernotificationservice.Service
	Logger          *zap.Logger
}

type Activities struct {
	credentialRepo  repositories.WorkerCredentialRepository
	credentials     *workercredentialservice.Service
	trainingRepo    repositories.WorkerTrainingRepository
	safetyRepo      repositories.WorkerSafetyRepository
	drugAlcoholRepo repositories.WorkerDrugAlcoholRepository
	training        *workertrainingservice.Service
	safety          *workersafetyservice.Service
	drugAlcohol     *workerdrugalcoholservice.Service
	dashControlRepo repositories.DashControlRepository
	notifications   *notificationservice.Service
	driverNotify    *drivernotificationservice.Service
	logger          *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		credentialRepo:  p.CredentialRepo,
		credentials:     p.Credentials,
		trainingRepo:    p.TrainingRepo,
		safetyRepo:      p.SafetyRepo,
		drugAlcoholRepo: p.DrugAlcoholRepo,
		training:        p.Training,
		safety:          p.Safety,
		drugAlcohol:     p.DrugAlcohol,
		dashControlRepo: p.DashControlRepo,
		notifications:   p.Notifications,
		driverNotify:    p.DriverNotify,
		logger:          p.Logger.Named("compliance-activities"),
	}
}

type sweepState struct {
	now            int64
	remindersByOrg map[pulid.ID]bool
	touchedWorkers map[pulid.ID]pagination.TenantInfo
	result         *CredentialExpirySweepResult
}

func (a *Activities) CredentialExpirySweepActivity(
	ctx context.Context,
) (*CredentialExpirySweepResult, error) {
	state := &sweepState{
		now:            timeutils.NowUnix(),
		remindersByOrg: make(map[pulid.ID]bool),
		touchedWorkers: make(map[pulid.ID]pagination.TenantInfo),
		result:         new(CredentialExpirySweepResult),
	}

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
			return nil, err
		}
		for _, cred := range page {
			state.result.CredentialsChecked++
			if err = a.sweepCredential(ctx, state, cred); err != nil {
				state.result.Failed++
				a.logger.Error("credential sweep failed",
					zap.String("credentialId", cred.ID.String()),
					zap.String("workerId", cred.WorkerID.String()),
					zap.Error(err))
			}
		}
		activity.RecordHeartbeat(ctx, state.result.CredentialsChecked)
		if len(page) < sweepPageSize {
			break
		}
		after = page[len(page)-1].ID
	}

	for workerID, tenantInfo := range state.touchedWorkers {
		state.result.WorkersChecked++
		if _, err := a.credentials.RefreshCompliance(ctx, tenantInfo, workerID); err != nil {
			a.logger.Warn("failed to refresh compliance after sweep",
				zap.String("workerId", workerID.String()),
				zap.Error(err))
		}
	}

	return state.result, nil
}

func (a *Activities) sweepCredential(
	ctx context.Context,
	state *sweepState,
	cred *worker.WorkerCredential,
) error {
	if cred.Worker == nil || cred.CredentialType == nil || cred.ExpiresAt == nil {
		return nil
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: cred.OrganizationID,
		BuID:  cred.BusinessUnitID,
	}
	state.touchedWorkers[cred.WorkerID] = tenantInfo

	daysLeft := worker.DaysUntil(*cred.ExpiresAt, state.now)
	if daysLeft > sweepHorizonDays || daysLeft < -sweepGraceDays {
		return nil
	}
	step := reminderStep(daysLeft)
	if step < 0 {
		return nil
	}

	remindDrivers, err := a.driverRemindersEnabled(ctx, tenantInfo, state.remindersByOrg)
	if err != nil {
		return err
	}
	if remindDrivers && !cred.Worker.UserID.IsNil() {
		sent, notifyErr := a.notifyDriver(ctx, tenantInfo, cred, daysLeft)
		if notifyErr != nil {
			return notifyErr
		}
		if sent {
			state.result.DriverNotifications++
		}
	}

	if daysLeft <= 0 {
		sent, alertErr := a.alertCompliance(ctx, tenantInfo, cred, daysLeft, true)
		if alertErr != nil {
			return alertErr
		}
		if sent {
			state.result.ComplianceAlerts++
		}
		return nil
	}

	if cred.CredentialType.IsRequired && step <= 14 {
		sent, alertErr := a.alertCompliance(ctx, tenantInfo, cred, daysLeft, false)
		if alertErr != nil {
			return alertErr
		}
		if sent {
			state.result.ComplianceAlerts++
		}
	}
	return nil
}

// reminderStep returns the escalation mark a days-left value sits on, or -1
// when the day is between marks. Expiry day and every day after are mark 0.
func reminderStep(daysLeft int64) int64 {
	if daysLeft <= 0 {
		return 0
	}
	for _, step := range reminderSteps {
		if daysLeft == step {
			return step
		}
	}
	return -1
}

// driverRemindersEnabled reports whether a driver should be told about this
// one credential now. A carrier on a daily or weekly digest still gets its
// reminders — bundled, by the digest pass — so sending one here as well would
// tell the driver twice.
func (a *Activities) driverRemindersEnabled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	cache map[pulid.ID]bool,
) (bool, error) {
	if enabled, ok := cache[tenantInfo.OrgID]; ok {
		return enabled, nil
	}
	control, err := a.dashControlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return false, err
	}
	enabled := control.SendCredentialReminders && !control.DriverDigestCadence.Bundles()
	cache[tenantInfo.OrgID] = enabled
	return enabled, nil
}

// credentialExpiryDate renders the expiry for the driver's message.
//
// The worker's own timezone is not on the credential, and a date is what a driver
// acts on rather than a clock time, so this formats the calendar date in UTC —
// which is the day the compliance sweep itself used to decide the credential was
// expiring.
func credentialExpiryDate(cred *worker.WorkerCredential) string {
	if cred.ExpiresAt == nil {
		return ""
	}
	return timeutils.FormatUnixDateIn(*cred.ExpiresAt, "")
}

func (a *Activities) alreadySent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	eventType, correlation string,
) (bool, error) {
	return a.notifications.ExistsRecent(
		ctx,
		repositories.ExistsRecentNotificationRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			EventType:      eventType,
			CorrelationID:  correlation,
			Since:          timeutils.NowUnix() - dedupeWindowDays*secondsPerDay,
		},
	)
}

func (a *Activities) notifyDriver(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	cred *worker.WorkerCredential,
	daysLeft int64,
) (bool, error) {
	correlation := fmt.Sprintf("cred-driver-%s", cred.ID)
	exists, err := a.alreadySent(ctx, tenantInfo, eventDriverExpiring, correlation)
	if err != nil || exists {
		return false, err
	}

	a.driverNotify.NotifyWithCorrelation(ctx, &drivernotificationservice.DriverNotification{
		TenantInfo: tenantInfo,
		WorkerID:   cred.WorkerID,
		EventType:  eventDriverExpiring,
		Priority:   notification.PriorityHigh,
		Context: documenttemplate.DriverNotificationContext{
			CredentialName: cred.CredentialType.Name,
			ExpiresInDays:  int(daysLeft),
			ExpiresAt:      credentialExpiryDate(cred),
		},
		Link: "/dash/profile",
	}, correlation)
	return true, nil
}

func (a *Activities) alertCompliance(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	cred *worker.WorkerCredential,
	daysLeft int64,
	expired bool,
) (bool, error) {
	eventType := eventComplianceExpiring
	if expired {
		eventType = eventComplianceExpired
	}
	correlation := fmt.Sprintf("cred-compliance-%s-%s", cred.ID, eventType)
	exists, err := a.alreadySent(ctx, tenantInfo, eventType, correlation)
	if err != nil || exists {
		return false, err
	}

	buID := tenantInfo.BuID
	correlationID := correlation
	name := cred.Worker.FirstName + " " + cred.Worker.LastName
	entity := &notification.Notification{
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  &buID,
		EventType:       eventType,
		Channel:         notification.ChannelGlobal,
		Data:            map[string]any{"link": complianceLink, "workerId": cred.WorkerID.String()},
		RelatedEntities: map[string]any{"workerId": cred.WorkerID.String(), "credentialId": cred.ID.String()},
		CorrelationID:   &correlationID,
		Source:          "compliance_sweep",
	}
	if expired {
		entity.Priority = notification.PriorityCritical
		entity.Title = "Driver credential expired"
		entity.Message = fmt.Sprintf(
			"%s's %s expired %d day(s) ago. The driver should not be dispatched until it is renewed.",
			name,
			cred.CredentialType.Name,
			-daysLeft,
		)
	} else {
		entity.Priority = notification.PriorityHigh
		entity.Title = "Driver credential expiring"
		entity.Message = fmt.Sprintf(
			"%s's %s expires in %d day(s) on %s. Schedule the renewal now.",
			name,
			cred.CredentialType.Name,
			daysLeft,
			credentialExpiryDate(cred),
		)
	}
	if _, err = a.notifications.Create(ctx, entity); err != nil {
		return false, err
	}
	return true, nil
}
