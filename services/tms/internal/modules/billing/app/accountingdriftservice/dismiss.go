package accountingdriftservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var dismissPermissions = []requiredPermission{
	{permission.ResourceAccountingSync, permission.OpUpdate},
}

func (s *Service) toleranceMinor(ctx context.Context, tenant pagination.TenantInfo) (int64, error) {
	if s.controls == nil {
		return 0, nil
	}
	control, err := s.controls.GetByOrgID(ctx, tenant.OrgID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return 0, nil
		}
		return 0, err
	}
	if control == nil || control.ReconciliationToleranceAmount.LessThanOrEqual(decimal.Zero) {
		return 0, nil
	}
	return money.MinorUnits(control.ReconciliationToleranceAmount), nil
}

func (s *Service) checkDismiss(
	ctx context.Context,
	finding *accountingdrift.AccountingDriftFinding,
	note string,
	actor *permissionports.RequestActor,
) (int64, error) {
	if !finding.IsOpen() {
		return 0, fixError(accountingdrift.ErrDriftClosed)
	}
	if stringutils.CollapseWhitespace(note) == "" {
		return 0, fixError(accountingdrift.ErrDriftNoteRequired)
	}
	tolerance, err := s.toleranceMinor(ctx, finding.TenantInfo())
	if err != nil {
		return 0, err
	}
	if actor.IsAgent() && !finding.WithinTolerance(tolerance) {
		return tolerance, errortypes.NewAuthorizationError(
			"An agent may dismiss only an amount difference within the reconciliation tolerance of {0}. A person needs to decide this one",
			money.FormatMinor(tolerance, finding.CurrencyCode),
		)
	}
	return tolerance, nil
}

func (s *Service) PreviewDismiss(
	ctx context.Context,
	req *accountingdriftports.DismissAccountingDriftRequest,
	actor *permissionports.RequestActor,
) (*accountingdriftports.AccountingDriftFixPreview, error) {
	if err := s.authorize(ctx, actor, dismissPermissions); err != nil {
		return nil, err
	}
	finding, err := s.findingByID(ctx, req.TenantInfo, req.ID, false)
	if err != nil {
		return nil, err
	}
	tolerance, err := s.checkDismiss(ctx, finding, req.Note, actor)
	if err != nil {
		return nil, err
	}
	return &accountingdriftports.AccountingDriftFixPreview{
		Finding:         finding,
		CurrencyCode:    finding.CurrencyCode,
		ToleranceMinor:  tolerance,
		WithinTolerance: finding.WithinTolerance(tolerance),
		Summary: "Dismiss " + describeFinding(
			finding,
		) + " and keep both sides as they are.",
	}, nil
}

func (s *Service) Dismiss(
	ctx context.Context,
	req *accountingdriftports.DismissAccountingDriftRequest,
	actor *permissionports.RequestActor,
) (*accountingdrift.AccountingDriftFinding, error) {
	if err := s.authorize(ctx, actor, dismissPermissions); err != nil {
		return nil, err
	}

	var updated *accountingdrift.AccountingDriftFinding
	var previous map[string]any
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		finding, txErr := s.findingByID(txCtx, req.TenantInfo, req.ID, true)
		if txErr != nil {
			return txErr
		}
		previous = jsonutils.MustToJSON(finding)
		if _, txErr = s.checkDismiss(txCtx, finding, req.Note, actor); txErr != nil {
			return txErr
		}
		if txErr = finding.Dismiss(actor.UserID, req.Note, s.now().Unix()); txErr != nil {
			return fixError(txErr)
		}
		updated, txErr = s.findings.Update(txCtx, finding)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(updated, actor.UserID, previous, "Dismissed a difference with the accounting system")
	s.refreshAttentionFor(ctx, updated)
	s.publishInvalidation(ctx, req.TenantInfo, actor.UserID, updated.ID)
	return updated, nil
}

func (s *Service) logAudit(
	finding *accountingdrift.AccountingDriftFinding,
	userID pulid.ID,
	previous map[string]any,
	comment string,
) {
	if s.audit == nil {
		return
	}
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceAccountingSync,
		ResourceID:     finding.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(finding),
		PreviousState:  previous,
		OrganizationID: finding.OrganizationID,
		BusinessUnitID: finding.BusinessUnitID,
	}
	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log accounting drift audit", zap.Error(err))
	}
}
