package workercredentialservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// eventCredentialRenewal is the driver-facing ask, worded by the
// organization's own template rather than here.
const eventCredentialRenewal = "dash.credential_renewal_requested"

// maxRenewalCredentials caps one ask. A driver with more papers than this
// coming due at once is a conversation, not a message.
const maxRenewalCredentials = 12

// RenewalRequest is one ask, covering every paper of a driver's that is
// coming due rather than one message per certificate.
type RenewalRequest struct {
	TenantInfo    pagination.TenantInfo
	WorkerID      pulid.ID
	CredentialIDs []pulid.ID
	Note          string
	RequestedByID pulid.ID
	// CorrelationID keys the ask so a retried proposal does not tell the
	// driver twice.
	CorrelationID string
}

/*
RequestRenewal asks a driver for the papers that are coming due.

It changes nothing on the credential: the renewal is the driver's to produce
and somebody's to verify, and recording one here that never arrived would
read as compliance the carrier does not have. What it does is put the ask on
the record, so the desk can tell an unanswered ask from one never made.
*/
func (s *Service) RequestRenewal(ctx context.Context, req RenewalRequest) error {
	named, err := s.planRenewal(ctx, &req)
	if err != nil {
		return err
	}

	s.notifyRenewal(ctx, req, named)
	s.auditRenewal(req, named)

	return nil
}

// planRenewal resolves the ask against the driver's own papers, within the
// bounds of one ask.
func (s *Service) planRenewal(
	ctx context.Context,
	req *RenewalRequest,
) ([]*worker.WorkerCredential, error) {
	if len(req.CredentialIDs) == 0 {
		return nil, errortypes.NewValidationError("credentialIds", errortypes.ErrRequired,
			"Name at least one credential to renew")
	}
	if len(req.CredentialIDs) > maxRenewalCredentials {
		return nil, errortypes.NewValidationError("credentialIds", errortypes.ErrInvalid,
			fmt.Sprintf("Ask for at most %d credentials at a time", maxRenewalCredentials))
	}

	held, err := s.repo.ListForWorker(ctx, &repositories.ListWorkerCredentialsRequest{
		TenantInfo:  req.TenantInfo,
		WorkerID:    req.WorkerID,
		IncludeType: true,
	})
	if err != nil {
		return nil, err
	}

	return s.namedCredentials(held, req.CredentialIDs)
}

// namedCredentials resolves the ask's ids against the driver's own papers,
// so an id belonging to somebody else is refused rather than silently
// included in a message about this driver.
func (s *Service) namedCredentials(
	held []*worker.WorkerCredential,
	wanted []pulid.ID,
) ([]*worker.WorkerCredential, error) {
	byID := make(map[pulid.ID]*worker.WorkerCredential, len(held))
	for _, credential := range held {
		if credential != nil {
			byID[credential.ID] = credential
		}
	}

	out := make([]*worker.WorkerCredential, 0, len(wanted))
	for _, id := range wanted {
		credential, ok := byID[id]
		if !ok {
			return nil, errortypes.NewValidationError("credentialIds", errortypes.ErrInvalid,
				fmt.Sprintf("Credential %s does not belong to this driver", id))
		}
		out = append(out, credential)
	}

	return out, nil
}

func (s *Service) notifyRenewal(
	ctx context.Context,
	req RenewalRequest,
	named []*worker.WorkerCredential,
) {
	if s.driverNotify == nil {
		return
	}

	notice, correlation := renewalNotification(&req, named, timeutils.NowUnix())
	s.driverNotify.NotifyWithCorrelation(ctx, notice, correlation)
}

// renewalNotification is the ask as the driver receives it: the papers by
// name, the soonest expiry, and the note, keyed so a retried ask is not sent
// twice.
func renewalNotification(
	req *RenewalRequest,
	named []*worker.WorkerCredential,
	now int64,
) (notice *drivernotificationservice.DriverNotification, correlation string) {
	soonest := named[0]
	for _, credential := range named {
		if credential.ExpiresAt == nil {
			continue
		}
		if soonest.ExpiresAt == nil || *credential.ExpiresAt < *soonest.ExpiresAt {
			soonest = credential
		}
	}

	daysLeft := 0
	expires := ""
	if soonest.ExpiresAt != nil {
		daysLeft = int(worker.DaysUntil(*soonest.ExpiresAt, now))
		expires = timeutils.FormatUnixDateIn(*soonest.ExpiresAt, "")
	}

	correlation = req.CorrelationID
	if strings.TrimSpace(correlation) == "" {
		correlation = "cred-renewal-" + req.WorkerID.String()
	}

	return &drivernotificationservice.DriverNotification{
		TenantInfo: req.TenantInfo,
		WorkerID:   req.WorkerID,
		EventType:  eventCredentialRenewal,
		Priority:   notification.PriorityHigh,
		Context: documenttemplate.DriverNotificationContext{
			CredentialName: renewalSubject(named),
			ExpiresInDays:  daysLeft,
			ExpiresAt:      expires,
			Reason:         req.Note,
		},
		Link: "/dash/profile",
	}, correlation
}

// renewalSubject names what is being asked for: the one paper when there is
// one, and how many when there are several.
func renewalSubject(named []*worker.WorkerCredential) string {
	if len(named) == 1 {
		if named[0].CredentialType != nil {
			return named[0].CredentialType.Name
		}

		return "a credential"
	}

	names := make([]string, 0, len(named))
	for _, credential := range named {
		if credential.CredentialType != nil {
			names = append(names, credential.CredentialType.Name)
		}
	}

	return strings.Join(names, ", ")
}

func (s *Service) auditRenewal(req RenewalRequest, named []*worker.WorkerCredential) {
	if s.auditService == nil {
		return
	}

	ids := make([]string, 0, len(named))
	for _, credential := range named {
		ids = append(ids, credential.ID.String())
	}

	if err := s.auditService.LogAction(&serviceports.LogActionParams{
		Resource:       permission.ResourceWorkerCredential,
		ResourceID:     req.WorkerID.String(),
		Operation:      permission.OpUpdate,
		UserID:         req.RequestedByID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		CurrentState: map[string]any{
			"credentialIds": ids,
			"note":          req.Note,
		},
	}, auditservice.WithComment(
		"Credential renewal requested: "+renewalSubject(named),
	)); err != nil {
		s.l.Warn("failed to audit credential renewal request")
	}
}
