package billingjobs

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const SendInvoiceEDIWorkflowName = "SendInvoiceEDIWorkflow"

type SendInvoiceEDIPayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID               `json:"invoiceId"`
	Force         bool                   `json:"force"`
	PrincipalType services.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID               `json:"principalId"`
	APIKeyID      pulid.ID               `json:"apiKeyId"`
}

type SendInvoiceEDIResult struct {
	InvoiceID   pulid.ID              `json:"invoiceId"`
	MessageID   pulid.ID              `json:"messageId"`
	Status      invoice.EDISendStatus `json:"status"`
	CompletedAt int64                 `json:"completedAt"`
}

// A 210 that fails validation will fail the same way every attempt, so it is
// not retried; an outage generating or queueing it is.
var sendInvoiceEDIRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    5,
	MaximumInterval:    time.Minute,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
		temporaltype.ErrorTypeNonRetryable.String(),
		temporaltype.ErrorTypePermissionDenied.String(),
	},
}

var sendInvoiceEDIActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         sendInvoiceEDIRetryPolicy,
}

// SendInvoiceEDIWorkflow generates the outbound 210 for a posted invoice and
// hands it to the partner's delivery channel.
func SendInvoiceEDIWorkflow(
	ctx workflow.Context,
	payload *SendInvoiceEDIPayload,
) (*SendInvoiceEDIResult, error) {
	ctx = workflow.WithActivityOptions(ctx, sendInvoiceEDIActivityOptions)

	var a *Activities
	result := new(SendInvoiceEDIResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.SendInvoiceEDIActivity,
		payload,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Invoice EDI send workflow failed", "error", err)
		return nil, err
	}

	return result, nil
}

// SendInvoiceEDIActivity resolves the invoice's EDI plan, generates the 210
// and records the outcome on the invoice. Delivery itself runs on the EDI
// task queue and writes Sending, Sent, Failed or DeadLettered back as it goes.
func (a *Activities) SendInvoiceEDIActivity(
	ctx context.Context,
	payload *SendInvoiceEDIPayload,
) (*SendInvoiceEDIResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}
	current, err := a.invoiceService.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         payload.InvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if current.Status != invoice.StatusPosted {
		return nil, temporal.NewNonRetryableApplicationError(
			"only a posted invoice can be sent by EDI",
			temporaltype.ErrorTypeInvalidInput.String(),
			nil,
		)
	}
	if !payload.Force && current.EDISendStatus == invoice.EDISendStatusSent {
		return &SendInvoiceEDIResult{
			InvoiceID:   current.ID,
			MessageID:   current.LastEDIMessageID,
			Status:      current.EDISendStatus,
			CompletedAt: timeutils.NowUnix(),
		}, nil
	}

	plans, err := a.invoiceService.ResolveEDISendPlans(ctx, &services.ResolveInvoiceEDISendPlansRequest{
		TenantInfo: tenantInfo,
		Invoices:   []*invoice.Invoice{current},
	})
	if err != nil {
		return nil, err
	}
	plan := plans[current.ID]
	if plan == nil || !plan.Enabled || len(plan.Blockers) > 0 {
		reason := "EDI invoicing is not configured for this customer"
		if plan != nil && len(plan.Blockers) > 0 {
			reason = plan.Blockers[0]
		}
		if err = a.invoiceRepo.UpdateEDISendStatus(ctx, repositories.UpdateInvoiceEDISendStatusRequest{
			TenantInfo: tenantInfo,
			InvoiceID:  current.ID,
			Status:     invoice.EDISendStatusNotConfigured,
			Error:      reason,
		}); err != nil {
			return nil, err
		}
		return &SendInvoiceEDIResult{
			InvoiceID:   current.ID,
			Status:      invoice.EDISendStatusNotConfigured,
			CompletedAt: timeutils.NowUnix(),
		}, nil
	}

	message, err := a.ediService.GenerateDocument(ctx, &services.GenerateEDIDocumentRequest{
		TenantInfo:               tenantInfo,
		PartnerDocumentProfileID: plan.DocumentProfileID,
		EDIPartnerID:             plan.PartnerID,
		InvoiceID:                current.ID,
		TransactionSet:           edi.TransactionSet210,
		Direction:                edi.DocumentDirectionOutbound,
		GeneratedByID:            payload.UserID,
	})
	if err != nil {
		if updateErr := a.invoiceRepo.UpdateEDISendStatus(ctx, repositories.UpdateInvoiceEDISendStatusRequest{
			TenantInfo: tenantInfo,
			InvoiceID:  current.ID,
			Status:     invoice.EDISendStatusFailed,
			Error:      err.Error(),
		}); updateErr != nil {
			a.logger.Warn("failed to record invoice EDI failure", zap.Error(updateErr))
		}
		if isInvalidEDIRequest(err) {
			return nil, temporal.NewNonRetryableApplicationError(
				err.Error(),
				temporaltype.ErrorTypeInvalidInput.String(),
				err,
			)
		}
		return nil, err
	}

	status := invoice.EDISendStatusGenerated
	var sentAt *int64
	if plan.CommunicationMethod == string(edi.ConnectionMethodInternal) {
		// An internal partner has no transport: the message is its own delivery.
		now := timeutils.NowUnix()
		status = invoice.EDISendStatusSent
		sentAt = &now
	}
	if err = a.invoiceRepo.UpdateEDISendStatus(ctx, repositories.UpdateInvoiceEDISendStatusRequest{
		TenantInfo: tenantInfo,
		InvoiceID:  current.ID,
		MessageID:  message.ID,
		Status:     status,
		SentAt:     sentAt,
	}); err != nil {
		return nil, err
	}

	return &SendInvoiceEDIResult{
		InvoiceID:   current.ID,
		MessageID:   message.ID,
		Status:      status,
		CompletedAt: timeutils.NowUnix(),
	}, nil
}

func isInvalidEDIRequest(err error) bool {
	var validationErr *errortypes.MultiError
	if errors.As(err, &validationErr) {
		return true
	}
	var typed *errortypes.Error
	if errors.As(err, &typed) {
		return typed.Code == errortypes.ErrInvalid || typed.Code == errortypes.ErrInvalidOperation
	}

	return false
}
