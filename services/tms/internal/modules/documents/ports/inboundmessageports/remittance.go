package inboundmessageports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type RemittanceIntakeAttachment struct {
	DocumentID  pulid.ID
	FileName    string
	ContentType string
}

type RemittanceIntakeRequest struct {
	TenantInfo  pagination.TenantInfo
	MessageID   pulid.ID
	FromAddress string
	ReceivedAt  int64
	Attachments []RemittanceIntakeAttachment
}

type RemittanceIntakeResult struct {
	FactoringCompanyID pulid.ID
	FactorName         string
	RemittanceIDs      []pulid.ID
	Problem            string
}

type RemittanceIntake interface {
	IngestRemittances(
		ctx context.Context,
		req *RemittanceIntakeRequest,
	) (*RemittanceIntakeResult, error)
}
