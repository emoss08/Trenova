package driverportalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
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

func (s *Service) MyPortalFeatures(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*PortalFeatures, error) {
	if _, err := s.ResolveWorker(ctx, tenantInfo); err != nil {
		return nil, err
	}
	control, err := s.dashControl(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return &PortalFeatures{
		RequireLoadAcknowledgment:  control.RequireLoadAcknowledgment,
		AllowLoadRefusals:          control.RequireLoadAcknowledgment && control.AllowLoadRefusals,
		AllowStopActions:           control.AllowStopActions,
		AllowLoadDocumentUpload:    control.AllowLoadDocumentUpload,
		AllowLoadComments:          control.AllowLoadComments,
		ShowLoadPay:                control.ShowLoadPay,
		ShowPayEstimates:           control.ShowLoadPay && control.ShowPayEstimates,
		AllowExpenseSubmission:     control.AllowExpenseSubmission,
		RequireExpenseReceipt:      control.AllowExpenseSubmission && control.RequireExpenseReceipt,
		AllowSettlementDisputes:    control.AllowSettlementDisputes,
		AllowProfileDocumentUpload: control.AllowProfileDocumentUpload,
		AllowContactInfoEdit:       control.AllowContactInfoEdit,
		AllowPtoRequests:           control.AllowPtoRequests,
	}, nil
}
