package factoringoverviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AgreementActivityRequest struct {
	TenantInfo  pagination.TenantInfo
	AgreementID pulid.ID
}

func (s *Service) AgreementActivity(
	ctx context.Context,
	req *AgreementActivityRequest,
) (*factoringports.AgreementActivity, error) {
	if req.AgreementID.IsNil() {
		return nil, errortypes.NewValidationError(
			"agreementId",
			errortypes.ErrRequired,
			"Choose the agreement whose activity to read",
		)
	}

	return s.repo.AgreementActivity(ctx, &factoringports.GetAgreementActivityRequest{
		TenantInfo:  req.TenantInfo,
		AgreementID: req.AgreementID,
	})
}
