package driverportalrecordsservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) MyPortalFeatures(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*PortalFeatures, error) {
	wrk, err := s.portal.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	control, err := s.portal.DashControl(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	ptoBalances := false
	if control.AllowPtoRequests && s.ptoLedger != nil {
		resolved, resolveErr := s.ptoLedger.ResolvePolicy(
			ctx,
			tenantInfo,
			wrk.ID,
			time.Now().Unix(),
		)
		if resolveErr != nil {
			return nil, resolveErr
		}
		ptoBalances = resolved != nil
	}
	leaveBalance, err := s.leaveVisible(ctx, tenantInfo, wrk.ID)
	if err != nil {
		return nil, err
	}
	schedule, err := s.scheduleVisible(ctx, tenantInfo, wrk.ID)
	if err != nil {
		return nil, err
	}
	outstanding, err := s.policiesOutstanding(ctx, tenantInfo, wrk)
	if err != nil {
		return nil, err
	}
	return &PortalFeatures{
		RequireLoadAcknowledgment: control.RequireLoadAcknowledgment,
		AllowLoadRefusals: control.RequireLoadAcknowledgment &&
			control.AllowLoadRefusals,
		AllowStopActions:        control.AllowStopActions,
		AllowLoadDocumentUpload: control.AllowLoadDocumentUpload,
		AllowLoadComments:       control.AllowLoadComments,
		ShowLoadPay:             control.ShowLoadPay,
		ShowPayEstimates:        control.ShowLoadPay && control.ShowPayEstimates,
		AllowExpenseSubmission:  control.AllowExpenseSubmission,
		RequireExpenseReceipt: control.AllowExpenseSubmission &&
			control.RequireExpenseReceipt,
		AllowSettlementDisputes:      control.AllowSettlementDisputes,
		AllowProfileDocumentUpload:   control.AllowProfileDocumentUpload,
		AllowContactInfoEdit:         control.AllowContactInfoEdit,
		AllowPtoRequests:             control.AllowPtoRequests,
		PtoBalances:                  ptoBalances,
		LeaveBalance:                 leaveBalance,
		Schedule:                     schedule,
		ShiftSwaps:                   schedule,
		RequireContactChangeApproval: control.RequireContactChangeApproval,
		PoliciesOutstanding:          outstanding,
	}, nil
}
