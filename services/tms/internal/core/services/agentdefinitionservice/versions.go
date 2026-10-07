package agentdefinitionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/editconflict"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type saveFunc func(ctx context.Context) (*agentdefinition.Definition, error)

// versioned writes the agent and the version it becomes in one step, so the
// history never names a save that did not happen nor misses one that did.
func (s *Service) versioned(
	write saveFunc,
	before *agentdefinition.Definition,
	actor *services.RequestActor,
) saveFunc {
	author := typeutils.IDPtr(actorUser(actor))
	return func(ctx context.Context) (*agentdefinition.Definition, error) {
		saved, err := write(ctx)
		if err != nil {
			return nil, err
		}
		if err = s.versions.Create(ctx, agentdefinition.NewVersion(saved, before, author)); err != nil {
			return nil, err
		}
		return saved, nil
	}
}

// inTransaction runs a save in a transaction of its own.
func (s *Service) inTransaction(ctx context.Context, write saveFunc) (*agentdefinition.Definition, error) {
	var saved *agentdefinition.Definition
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var txErr error
		saved, txErr = write(txCtx)
		return txErr
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

type conflictRequest struct {
	tenantInfo pagination.TenantInfo
	agentID    pulid.ID
	loaded     int64
	cause      error
}

// editConflict turns a save that lost a race into what the person needs to
// settle it: who saved in between, when, and which settings they changed
// since the version this person loaded. Anything short of that still reports
// the conflict, without the detail it could not find.
func (s *Service) editConflict(ctx context.Context, req *conflictRequest) error {
	if !errortypes.IsVersionMismatchError(req.cause) {
		return req.cause
	}

	current, err := s.repo.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.agentID,
		TenantInfo: req.tenantInfo,
	})
	if err != nil {
		return req.cause
	}

	detail := &editconflict.Detail{Version: current.Version, UpdatedAt: current.UpdatedAt}

	latest, err := s.versions.LatestAt(ctx, &repositories.GetAgentDefinitionVersionAtRequest{
		TenantInfo:        req.tenantInfo,
		AgentDefinitionID: req.agentID,
		Version:           current.Version,
	})
	if err != nil {
		s.l.Warn("could not read who last saved the agent", zap.Error(err))
	}
	if latest != nil {
		detail.Author = latest.Author
	}

	loaded, err := s.versions.LatestAt(ctx, &repositories.GetAgentDefinitionVersionAtRequest{
		TenantInfo:        req.tenantInfo,
		AgentDefinitionID: req.agentID,
		Version:           req.loaded,
	})
	if err != nil {
		s.l.Warn("could not read the version the person loaded", zap.Error(err))
	}
	if loaded != nil {
		detail.Changes = agentdefinition.Changes(loaded.Snapshot, current)
	}

	return editconflict.Error(detail, req.cause)
}

func (s *Service) ListVersions(
	ctx context.Context,
	req *repositories.ListAgentDefinitionVersionsRequest,
) ([]*agentdefinition.DefinitionVersion, error) {
	if _, err := s.repo.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	return s.versions.List(ctx, req)
}

// RestoreVersion loads an earlier version as a draft of the agent as it is
// now: its settings, on the current version number, so saving the draft
// replaces the agent the way any edit does. Nothing is saved here.
func (s *Service) RestoreVersion(
	ctx context.Context,
	req *repositories.GetAgentDefinitionVersionRequest,
) (*agentdefinition.Definition, error) {
	current, err := s.repo.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	version, err := s.versions.Get(ctx, req)
	if err != nil {
		return nil, err
	}
	if version.Snapshot == nil {
		return nil, errortypes.NewNotFoundError("Agent version")
	}

	draft := *version.Snapshot
	draft.ID = current.ID
	draft.BusinessUnitID = current.BusinessUnitID
	draft.OrganizationID = current.OrganizationID
	draft.SystemKey = current.SystemKey
	draft.CreatedByID = current.CreatedByID
	draft.AccessMode = current.AccessMode
	draft.LastRunAt = current.LastRunAt
	draft.NextRunAt = current.NextRunAt
	draft.Version = current.Version
	draft.CreatedAt = current.CreatedAt
	draft.UpdatedAt = current.UpdatedAt

	return &draft, nil
}
