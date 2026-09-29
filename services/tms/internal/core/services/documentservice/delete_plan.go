package documentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DeletePlan struct {
	Documents []*document.Document
	Versions  map[pulid.ID][]*document.Document
}

func (p *DeletePlan) FileCount() int {
	count := 0
	for _, doc := range p.Documents {
		if versions, ok := p.Versions[doc.LineageID]; ok && doc.LineageID.IsNotNil() {
			count += len(versions)
			continue
		}
		count++
	}

	return count
}

type RestorePlan struct {
	Target    *document.Document
	Current   *document.Document
	Unchanged bool
}

func (s *Service) PlanDelete(ctx context.Context, req *BulkDeleteRequest) (*DeletePlan, error) {
	if len(req.IDs) == 0 {
		return &DeletePlan{Versions: map[pulid.ID][]*document.Document{}}, nil
	}

	docs, err := s.repo.GetByIDs(ctx, repositories.BulkDeleteDocumentRequest{
		IDs:        req.IDs,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	found := make(map[pulid.ID]bool, len(docs))
	for _, doc := range docs {
		found[doc.ID] = true
	}
	missing := make([]string, 0)
	for _, id := range req.IDs {
		if !found[id] {
			missing = append(missing, id.String())
		}
	}
	if len(missing) > 0 {
		return nil, errortypes.NewValidationError("ids", errortypes.ErrInvalid,
			"These are not documents of this organization: {0}", strings.Join(missing, ", "))
	}

	plan := &DeletePlan{
		Documents: docs,
		Versions:  make(map[pulid.ID][]*document.Document, len(docs)),
	}
	for _, doc := range docs {
		if doc.LineageID.IsNil() {
			continue
		}
		if _, listed := plan.Versions[doc.LineageID]; listed {
			continue
		}
		versions, versionErr := s.repo.ListVersions(ctx, repositories.ListDocumentVersionsRequest{
			LineageID:  doc.LineageID,
			TenantInfo: req.TenantInfo,
		})
		if versionErr != nil {
			return nil, versionErr
		}
		plan.Versions[doc.LineageID] = versions
	}

	return plan, nil
}

func (s *Service) PlanRestoreVersion(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*RestorePlan, error) {
	target, err := s.repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	versions, err := s.repo.ListVersions(ctx, repositories.ListDocumentVersionsRequest{
		LineageID:  target.LineageID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	current := currentDocumentVersion(versions)

	return &RestorePlan{
		Target:    target,
		Current:   current,
		Unchanged: current != nil && current.ID == target.ID,
	}, nil
}
