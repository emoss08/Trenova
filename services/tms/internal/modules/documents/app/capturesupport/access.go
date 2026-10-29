// Package capturesupport is what every Trenova Capture service shares: the
// organization's capture switch, the permission check, realtime invalidation,
// the audit trail and the stored page objects.
//
// The companion never decides anything. It captures pages and uploads them;
// where a stack divides, where each part goes, and whether it may be filed
// without a person are all decided here, so the rules are the same for every
// device and can change without shipping a new install.
package capturesupport

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// ErrCaptureDisabled is what every capture call gets in an organization that
// turned capture off. It is a refusal, not a failure: the device keeps its
// pages and tells the person why.
var ErrCaptureDisabled = errors.New("document capture is turned off for this organization")

// DisabledReasonParam and DisabledReason tag ErrCaptureDisabled in the problem
// body, so the companion can tell "hold the pages until capture is turned
// back on" from a refusal of the upload itself without reading its text.
const (
	DisabledReasonParam = "reason"
	DisabledReason      = "capture_disabled"
)

// control is the tenant's document settings. A tenant that never saved them
// gets the defaults, which is what it would have been given on creation.
func (s *Service) Control(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*documentcontrol.DocumentControl, error) {
	control, err := s.controls.Get(ctx, tenantports.GetDocumentControlRequest{
		TenantInfo: tenantInfo,
	})
	if err == nil {
		return control, nil
	}
	if errortypes.IsNotFoundError(err) {
		return documentcontrol.NewDefaultDocumentControl(tenantInfo.OrgID, tenantInfo.BuID), nil
	}

	return nil, err
}

// requireEnabled refuses when the tenant turned capture off.
func (s *Service) RequireEnabled(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*documentcontrol.DocumentControl, error) {
	control, err := s.Control(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !control.EnableCapture {
		return nil, errortypes.NewBusinessError(ErrCaptureDisabled.Error()).
			WithParam(DisabledReasonParam, DisabledReason).
			WithInternal(ErrCaptureDisabled)
	}

	return control, nil
}

// allowed reports whether the person may do op on resource, and at what data
// scope. It is the one place capture asks, so a device and a browser session
// acting for the same person always get the same answer.
func (s *Service) allowed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource permission.Resource,
	op permission.Operation,
) (*permissionports.PermissionCheckResult, error) {
	if tenantInfo.UserID.IsNil() {
		return &permissionports.PermissionCheckResult{Allowed: false, Reason: "no_user"}, nil
	}

	return s.permissions.Check(
		ctx,
		permissionports.UserActor(tenantInfo).PermissionCheck(resource, op),
	)
}

// require is allowed, turned into the error a handler maps to 403.
func (s *Service) Require(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource permission.Resource,
	op permission.Operation,
) (*permissionports.PermissionCheckResult, error) {
	result, err := s.allowed(ctx, tenantInfo, resource, op)
	if err != nil {
		return nil, err
	}
	if !result.Allowed {
		return nil, errortypes.NewAuthorizationError(
			"You do not have permission to " + string(op) + " " + resource.String(),
		)
	}

	return result, nil
}

// seesEveryone reports whether a permission result reaches past the person's
// own records.
func SeesEveryone(result *permissionports.PermissionCheckResult) bool {
	return result != nil && result.DataScope != permission.DataScopeOwn
}

// publish tells open screens that a capture record changed. It is addressed
// to one person when audience is set, which is how a device's stream learns
// of a request without the rest of the tenant hearing about it.
func (s *Service) Publish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource permission.Resource,
	recordID pulid.ID,
	action string,
	audience pulid.ID,
	entity any,
) {
	if s.realtime == nil {
		return
	}

	if err := s.realtime.PublishResourceInvalidation(
		ctx,
		&realtimeports.PublishResourceInvalidationRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			AudienceUserID: audience,
			Resource:       resource.String(),
			Action:         action,
			RecordID:       recordID,
			Entity:         entity,
			ActorUserID:    tenantInfo.UserID,
			ActorType:      permissionports.PrincipalTypeUser,
			ActorID:        tenantInfo.UserID,
		},
	); err != nil {
		s.l.Warn("capture invalidation lost",
			zap.String("resource", resource.String()),
			zap.String("recordId", recordID.String()),
			zap.Error(err))
	}
}

// logAudit records an action in the audit trail. A failure to record is
// logged, never returned: the action already happened.
func (s *Service) LogAudit(params *auditports.LogActionParams, comment string) {
	if s.audit == nil {
		return
	}

	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to record capture audit entry",
			zap.String("resource", params.Resource.String()),
			zap.String("resourceId", params.ResourceID),
			zap.Error(err))
	}
}

// Access is what the web app needs to decide whether to offer capture at
// all: whether the organization turned it on, and whether this person may
// use it. Neither is a secret, and asking needs no permission beyond being
// signed in, so a record's page can hide the scan button rather than show
// one that fails.
type Access struct {
	Enabled    bool `json:"enabled"`
	CanCapture bool `json:"canCapture"`
}

func (s *Service) Access(ctx context.Context, tenantInfo pagination.TenantInfo) (*Access, error) {
	control, err := s.Control(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	result, err := s.allowed(ctx, tenantInfo, permission.ResourceCaptureBatch, permission.OpCreate)
	if err != nil {
		return nil, err
	}

	return &Access{Enabled: control.EnableCapture, CanCapture: result.Allowed}, nil
}
