package recordshareservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/recordshareports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	candidateScanPageSize = 100
	maxCandidateScan      = 2000
	defaultCandidates     = 10
	maxCandidates         = 25
)

// ListCandidates reads the organization's people a page at a time and keeps
// those the link would open for, stopping as soon as it has what was asked
// for: the picker shows a handful, so it never scans the whole directory to
// fill one list.
func (s *Service) ListCandidates(
	ctx context.Context,
	req *recordshareports.CandidatesRequest,
	actor *permissionports.RequestActor,
) ([]*tenant.User, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	record, err := s.record(ctx, req.TenantInfo, req.Kind, req.RecordID, actor)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultCandidates
	}
	limit = min(limit, maxCandidates)

	resource := record.Kind.Resource()
	eligible := make([]*tenant.User, 0, limit)
	for offset := 0; offset < maxCandidateScan; offset += candidateScanPageSize {
		page, pErr := s.userRepo.SelectOptions(ctx, &pagination.SelectQueryRequest{
			TenantInfo: req.TenantInfo,
			Query:      req.Query,
			Pagination: pagination.Info{Limit: candidateScanPageSize, Offset: offset},
		})
		if pErr != nil {
			return nil, pErr
		}

		for _, user := range page.Items {
			if user == nil || user.ID == actor.UserID || !isShareableUser(user) {
				continue
			}
			ok, checkErr := s.canOpen(ctx, req.TenantInfo, resource, user)
			if checkErr != nil {
				return nil, checkErr
			}
			if !ok {
				continue
			}
			eligible = append(eligible, user)
			if len(eligible) == limit {
				return eligible, nil
			}
		}

		if len(page.Items) < candidateScanPageSize ||
			(page.TotalExact && offset+candidateScanPageSize >= page.Total) {
			break
		}
	}

	return eligible, nil
}
