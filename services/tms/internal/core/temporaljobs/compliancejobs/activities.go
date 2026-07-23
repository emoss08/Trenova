package compliancejobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	sweepHorizonDays = 30
	sweepGraceDays   = 7
	dedupeWindowDays = 6
	secondsPerDay    = int64(86400)
)

// reminderSteps are the days-until-expiry marks at which a driver is reminded.
var reminderSteps = []int64{30, 14, 3}

type ActivitiesParams struct {
	fx.In

	WorkerRepo      repositories.WorkerRepository
	DashControlRepo repositories.DashControlRepository
	Notifications   *notificationservice.Service
	DriverNotify    *drivernotificationservice.Service
	Logger          *zap.Logger
}

type Activities struct {
	workerRepo      repositories.WorkerRepository
	dashControlRepo repositories.DashControlRepository
	notifications   *notificationservice.Service
	driverNotify    *drivernotificationservice.Service
	logger          *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		workerRepo:      p.WorkerRepo,
		dashControlRepo: p.DashControlRepo,
		notifications:   p.Notifications,
		driverNotify:    p.DriverNotify,
		logger:          p.Logger.Named("compliance-activities"),
	}
}

type credential struct {
	Name   string
	Expiry *int64
}

func workerCredentials(profile *worker.WorkerProfile) []credential {
	if profile == nil {
		return nil
	}
	licenseExpiry := profile.LicenseExpiry
	return []credential{
		{Name: "CDL", Expiry: &licenseExpiry},
		{Name: "Hazmat endorsement", Expiry: profile.HazmatExpiry},
		{Name: "Medical card", Expiry: profile.MedicalCardExpiry},
		{Name: "DOT physical", Expiry: profile.PhysicalDueDate},
		{Name: "MVR review", Expiry: profile.MVRDueDate},
		{Name: "TWIC card", Expiry: profile.TWICExpiry},
	}
}

func (a *Activities) CredentialExpirySweepActivity(
	ctx context.Context,
) (*CredentialExpirySweepResult, error) {
	result := new(CredentialExpirySweepResult)

	workers, err := a.workerRepo.ListWorkersWithExpiringCredentials(
		ctx,
		repositories.ListExpiringCredentialsRequest{
			HorizonDays: sweepHorizonDays,
			GraceDays:   sweepGraceDays,
		},
	)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	remindersByOrg := make(map[pulid.ID]bool)
	for _, wrk := range workers {
		result.WorkersChecked++
		if err = a.sweepWorker(ctx, now, wrk, remindersByOrg, result); err != nil {
			result.Failed++
			a.logger.Error("credential sweep failed for worker",
				zap.String("workerId", wrk.ID.String()),
				zap.Error(err))
		}
	}
	return result, nil
}

func (a *Activities) sweepWorker(
	ctx context.Context,
	now int64,
	wrk *worker.Worker,
	remindersByOrg map[pulid.ID]bool,
	result *CredentialExpirySweepResult,
) error {
	tenantInfo := pagination.TenantInfo{
		OrgID: wrk.OrganizationID,
		BuID:  wrk.BusinessUnitID,
	}
	remindDrivers, err := a.driverRemindersEnabled(ctx, tenantInfo, remindersByOrg)
	if err != nil {
		return err
	}

	for _, cred := range workerCredentials(wrk.Profile) {
		if cred.Expiry == nil || *cred.Expiry == 0 {
			continue
		}
		daysLeft := (*cred.Expiry - now) / secondsPerDay
		if daysLeft > sweepHorizonDays || daysLeft < -sweepGraceDays {
			continue
		}

		if remindDrivers && a.shouldRemindDriver(daysLeft) && !wrk.UserID.IsNil() {
			sent, notifyErr := a.notifyDriver(ctx, tenantInfo, wrk, cred, daysLeft)
			if notifyErr != nil {
				return notifyErr
			}
			if sent {
				result.DriverNotifications++
			}
		}

		if daysLeft <= 0 {
			sent, alertErr := a.alertCompliance(ctx, tenantInfo, wrk, cred, daysLeft)
			if alertErr != nil {
				return alertErr
			}
			if sent {
				result.ComplianceAlerts++
			}
		}
	}
	return nil
}

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
	cache[tenantInfo.OrgID] = control.SendCredentialReminders
	return control.SendCredentialReminders, nil
}

func (a *Activities) shouldRemindDriver(daysLeft int64) bool {
	if daysLeft <= 0 {
		return true
	}
	for _, step := range reminderSteps {
		if daysLeft <= step && daysLeft > step-1 {
			return true
		}
	}
	return false
}

func (a *Activities) notifyDriver(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	wrk *worker.Worker,
	cred credential,
	daysLeft int64,
) (bool, error) {
	correlation := fmt.Sprintf("cred-driver-%s-%s", wrk.ID, cred.Name)
	exists, err := a.notifications.ExistsRecent(
		ctx,
		repositories.ExistsRecentNotificationRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			EventType:      "dash.credential_expiring",
			CorrelationID:  correlation,
			Since:          timeutils.NowUnix() - dedupeWindowDays*secondsPerDay,
		},
	)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	message := fmt.Sprintf(
		"Your %s expires in %d days. Upload the renewed document in Dash to stay road-ready.",
		cred.Name,
		daysLeft,
	)
	if daysLeft <= 0 {
		message = fmt.Sprintf(
			"Your %s has expired. Upload the renewed document in Dash right away — you can't be dispatched until it's current.",
			cred.Name,
		)
	}
	a.driverNotify.NotifyWithCorrelation(ctx, &drivernotificationservice.DriverNotification{
		TenantInfo: tenantInfo,
		WorkerID:   wrk.ID,
		EventType:  "dash.credential_expiring",
		Priority:   notification.PriorityHigh,
		Title:      cred.Name + " renewal needed",
		Message:    message,
		Link:       "/dash/profile",
	}, correlation)
	return true, nil
}

func (a *Activities) alertCompliance(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	wrk *worker.Worker,
	cred credential,
	daysLeft int64,
) (bool, error) {
	correlation := fmt.Sprintf("cred-compliance-%s-%s", wrk.ID, cred.Name)
	exists, err := a.notifications.ExistsRecent(
		ctx,
		repositories.ExistsRecentNotificationRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			EventType:      "credential_expired",
			CorrelationID:  correlation,
			Since:          timeutils.NowUnix() - dedupeWindowDays*secondsPerDay,
		},
	)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	buID := tenantInfo.BuID
	correlationID := correlation
	name := wrk.FirstName + " " + wrk.LastName
	entity := &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		EventType:      "credential_expired",
		Priority:       notification.PriorityCritical,
		Channel:        notification.ChannelGlobal,
		Title:          "Driver credential expired",
		Message: fmt.Sprintf(
			"%s's %s expired %d day(s) ago. The driver should not be dispatched until it is renewed.",
			name,
			cred.Name,
			-daysLeft,
		),
		Data: map[string]any{"link": "/dispatch-management/workers"},
		RelatedEntities: map[string]any{
			"workerId": wrk.ID.String(),
		},
		CorrelationID: &correlationID,
		Source:        "compliance_sweep",
	}
	if _, err = a.notifications.Create(ctx, entity); err != nil {
		return false, err
	}
	return true, nil
}
