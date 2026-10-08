package aiproviderservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const priorityStep = 10

type change struct {
	saved    *aiprovider.Provider
	previous *aiprovider.Provider
}

func (s *Service) Reorder(
	ctx context.Context,
	req *services.ReorderAIProvidersRequest,
	actor *services.RequestActor,
) ([]*aiprovider.Provider, error) {
	current, err := s.repo.ListOrdered(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if !sameProviders(current, req.ProviderIDs) {
		return nil, errortypes.NewValidationError(
			"providerIds", errortypes.ErrInvalid,
			"The order must name every provider once; reload the list and try again",
		)
	}

	priorities := make(map[pulid.ID]int, len(req.ProviderIDs))
	for idx, id := range req.ProviderIDs {
		priorities[id] = (idx + 1) * priorityStep
	}

	changes := make([]change, 0, len(current))
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		for _, provider := range current {
			next := priorities[provider.ID]
			if provider.Priority == next {
				continue
			}
			saved, previous, modifyErr := s.modify(txCtx, &modifyRequest{
				tenantInfo: req.TenantInfo,
				providerID: provider.ID,
				actor:      actor,
				edit:       func(p *aiprovider.Provider) { p.Priority = next },
			})
			if modifyErr != nil {
				return modifyErr
			}
			changes = append(changes, change{saved: saved, previous: previous})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for idx := range changes {
		s.logAudit(&auditParams{
			provider:  changes[idx].saved,
			previous:  changes[idx].previous,
			operation: permission.OpUpdate,
			actor:     actor,
			comment:   "AI provider moved in the routing order",
		})
	}

	return s.ordered(ctx, req.TenantInfo)
}

func (s *Service) AssignTask(
	ctx context.Context,
	req *services.AssignAIProviderTaskRequest,
	actor *services.RequestActor,
) (*aiprovider.Provider, error) {
	if !req.Task.IsValid() {
		return nil, errortypes.NewValidationError("task", errortypes.ErrInvalid, "Unknown task")
	}

	saved, previous, err := s.modify(ctx, &modifyRequest{
		tenantInfo: req.TenantInfo,
		providerID: req.ProviderID,
		actor:      actor,
		edit: func(p *aiprovider.Provider) {
			if !p.ServesTask(req.Task) {
				p.Tasks = append(slices.Clone(p.Tasks), req.Task)
			}
		},
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(&auditParams{
		provider:  saved,
		previous:  previous,
		operation: permission.OpUpdate,
		actor:     actor,
		comment:   "Task assigned to AI provider",
	})

	return saved.Redacted(), nil
}

type modifyRequest struct {
	tenantInfo pagination.TenantInfo
	providerID pulid.ID
	// version is the version the editor loaded; nil takes the stored one.
	version *int64
	actor   *services.RequestActor
	edit    func(*aiprovider.Provider)
}

func (s *Service) modify(
	ctx context.Context,
	req *modifyRequest,
) (*aiprovider.Provider, *aiprovider.Provider, error) {
	existing, err := s.repo.GetByID(ctx, repositories.GetAIProviderByIDRequest{
		ID:         req.providerID,
		TenantInfo: req.tenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	previous := existing.Redacted()
	updated := *existing
	updated.Tasks = slices.Clone(existing.Tasks)
	loaded := existing.Version
	if req.version != nil {
		loaded = *req.version
		updated.Version = *req.version
	}
	req.edit(&updated)

	multiErr := errortypes.NewMultiError()
	updated.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, nil, multiErr
	}

	saved, err := s.save(ctx, func(txCtx context.Context) (*aiprovider.Provider, error) {
		return s.repo.Update(txCtx, &updated)
	}, req.actor)
	if err != nil {
		return nil, nil, s.explainConflict(ctx, &conflictRequest{
			tenantInfo: req.tenantInfo,
			providerID: req.providerID,
			loaded:     loaded,
			cause:      err,
		})
	}
	return saved, previous, nil
}

func (s *Service) ordered(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*aiprovider.Provider, error) {
	providers, err := s.repo.ListOrdered(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	for idx, provider := range providers {
		providers[idx] = provider.Redacted()
	}
	return providers, nil
}

func sameProviders(current []*aiprovider.Provider, ids []pulid.ID) bool {
	if len(current) != len(ids) {
		return false
	}
	want := make(map[pulid.ID]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := want[id]; dup {
			return false
		}
		want[id] = struct{}{}
	}
	for _, provider := range current {
		if _, ok := want[provider.ID]; !ok {
			return false
		}
	}
	return true
}
