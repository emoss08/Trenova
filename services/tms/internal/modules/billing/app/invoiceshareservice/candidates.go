package invoiceshareservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	candidateScanPageSize = 100
	maxCandidateScan      = 2000
)

func (s *Service) ListCandidates(
	ctx context.Context,
	req *invoiceports.ListShareCandidatesRequest,
) (*pagination.ListResult[*invoiceports.ShareCandidate], error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	if _, err := s.invoiceRepo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	eligible, err := s.eligibleCandidates(ctx, req.TenantInfo, req.Query)
	if err != nil {
		return nil, err
	}

	limit := req.Pagination.SafeLimit()
	offset := min(req.Pagination.SafeOffset(), len(eligible))
	end := min(offset+limit, len(eligible))

	return &pagination.ListResult[*invoiceports.ShareCandidate]{
		Items: eligible[offset:end],
		Total: len(eligible),
	}, nil
}

func (s *Service) eligibleCandidates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query string,
) ([]*invoiceports.ShareCandidate, error) {
	eligible := make([]*invoiceports.ShareCandidate, 0)

	for offset := 0; offset < maxCandidateScan; offset += candidateScanPageSize {
		page, err := s.userRepo.SelectOptions(ctx, &pagination.SelectQueryRequest{
			TenantInfo: tenantInfo,
			Query:      query,
			Pagination: pagination.Info{Limit: candidateScanPageSize, Offset: offset},
		})
		if err != nil {
			return nil, err
		}

		for _, user := range page.Items {
			ok, checkErr := s.isCandidate(ctx, tenantInfo, user)
			if checkErr != nil {
				return nil, checkErr
			}
			if ok {
				eligible = append(eligible, toShareCandidate(user))
			}
		}

		if len(page.Items) < candidateScanPageSize || offset+candidateScanPageSize >= page.Total {
			break
		}
	}

	return eligible, nil
}

func (s *Service) GetCandidate(
	ctx context.Context,
	req *invoiceports.GetShareCandidateRequest,
) (*invoiceports.ShareCandidate, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	if _, err := s.invoiceRepo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:   req.TenantInfo,
		LookupUserID: req.UserID,
	})
	if err != nil {
		return nil, err
	}

	ok, err := s.isCandidate(ctx, req.TenantInfo, user)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errortypes.NewNotFoundError("Teammate not found")
	}

	return toShareCandidate(user), nil
}

func (s *Service) isCandidate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	user *tenant.User,
) (bool, error) {
	if user == nil || user.ID == tenantInfo.UserID || !isShareableUser(user) {
		return false, nil
	}

	return s.canViewInvoices(ctx, tenantInfo, user)
}

func toShareCandidate(user *tenant.User) *invoiceports.ShareCandidate {
	return &invoiceports.ShareCandidate{
		ID:            user.ID,
		Name:          user.Name,
		Username:      user.Username,
		EmailAddress:  user.EmailAddress,
		ProfilePicURL: user.ProfilePicURL,
		ThumbnailURL:  user.ThumbnailURL,
	}
}
