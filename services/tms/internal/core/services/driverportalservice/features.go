package driverportalservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/services/capabilityguard"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

// PortalFeatures is the driver-facing view of the org's DashControl: which
// portal capabilities the carrier has enabled for its drivers.
type PortalFeatures struct {
	RequireLoadAcknowledgment  bool `json:"requireLoadAcknowledgment"`
	AllowLoadRefusals          bool `json:"allowLoadRefusals"`
	AllowStopActions           bool `json:"allowStopActions"`
	AllowLoadDocumentUpload    bool `json:"allowLoadDocumentUpload"`
	AllowLoadComments          bool `json:"allowLoadComments"`
	ShowLoadPay                bool `json:"showLoadPay"`
	ShowPayEstimates           bool `json:"showPayEstimates"`
	AllowExpenseSubmission     bool `json:"allowExpenseSubmission"`
	RequireExpenseReceipt      bool `json:"requireExpenseReceipt"`
	AllowSettlementDisputes    bool `json:"allowSettlementDisputes"`
	AllowProfileDocumentUpload bool `json:"allowProfileDocumentUpload"`
	AllowContactInfoEdit       bool `json:"allowContactInfoEdit"`
	AllowPtoRequests           bool `json:"allowPtoRequests"`
	PtoBalances                bool `json:"ptoBalances"`
	// LeaveBalance is on only once the driver has a leave case. An empty FMLA
	// card reads as an entitlement they are owed rather than one they have not
	// started drawing on.
	LeaveBalance bool `json:"leaveBalance"`
	// Schedule is on once the driver is actually on a shift. An empty rota
	// reads as a roster nobody filled in rather than as a job with no fixed
	// pattern, which is what an owner-operator has.
	Schedule bool `json:"schedule"`
	// ShiftSwaps follows Schedule: a day you are not rostered on is not a day
	// you can offer anybody.
	ShiftSwaps bool `json:"shiftSwaps"`
	// RequireContactChangeApproval is surfaced so the app can say "this will
	// be sent to your carrier" before the driver saves, not after.
	RequireContactChangeApproval bool `json:"requireContactChangeApproval"`
	// PoliciesOutstanding is how many signatures the driver still owes.
	PoliciesOutstanding int `json:"policiesOutstanding"`
}

func (s *Service) dashControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.DashControl, error) {
	return s.dashControlRepo.GetOrCreate(ctx, tenantInfo)
}

// requireFeature loads the org's DashControl and returns a validation error
// when the given capability is switched off for drivers.
func (s *Service) requireFeature(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	enabled func(*tenant.DashControl) bool,
	message string,
) (*tenant.DashControl, error) {
	control, err := s.dashControl(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !enabled(control) {
		return nil, errortypes.NewValidationError(
			"feature",
			errortypes.ErrInvalidOperation,
			message,
		)
	}
	return control, nil
}

func (s *Service) requireAssetOperations(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	enabled, err := capabilityguard.AssetOperationsEnabled(ctx, s.orgRepo, tenantInfo)
	if err != nil {
		return err
	}
	if !enabled {
		return errortypes.NewValidationError(
			"feature",
			errortypes.ErrInvalidOperation,
			"Driver portal access requires asset operations. Enable asset operations for this organization before inviting drivers",
		)
	}

	return nil
}

func (s *Service) MyPortalFeatures(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*PortalFeatures, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	control, err := s.dashControl(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	ptoBalances := false
	if control.AllowPtoRequests && s.ptoLedger != nil {
		resolved, resolveErr := s.ptoLedger.ResolvePolicy(ctx, tenantInfo, wrk.ID, time.Now().Unix())
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
		RequireLoadAcknowledgment:    control.RequireLoadAcknowledgment,
		AllowLoadRefusals:            control.RequireLoadAcknowledgment && control.AllowLoadRefusals,
		AllowStopActions:             control.AllowStopActions,
		AllowLoadDocumentUpload:      control.AllowLoadDocumentUpload,
		AllowLoadComments:            control.AllowLoadComments,
		ShowLoadPay:                  control.ShowLoadPay,
		ShowPayEstimates:             control.ShowLoadPay && control.ShowPayEstimates,
		AllowExpenseSubmission:       control.AllowExpenseSubmission,
		RequireExpenseReceipt:        control.AllowExpenseSubmission && control.RequireExpenseReceipt,
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
