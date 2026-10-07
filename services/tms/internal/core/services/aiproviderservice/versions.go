package aiproviderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/settingversion"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/editconflict"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type writeFunc func(ctx context.Context) (*aiprovider.Provider, error)

// save writes the provider and the version it becomes together. The version
// keeps the provider without its key, which never leaves the provider row.
func (s *Service) save(
	ctx context.Context,
	write writeFunc,
	actor *services.RequestActor,
) (*aiprovider.Provider, error) {
	var saved *aiprovider.Provider
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var txErr error
		saved, txErr = write(txCtx)
		if txErr != nil {
			return txErr
		}
		return editconflict.Record(txCtx, s.versions, &editconflict.RecordRequest{
			TenantInfo: pagination.TenantInfo{OrgID: saved.OrganizationID, BuID: saved.BusinessUnitID},
			Kind:       settingversion.KindAIProvider,
			SubjectID:  saved.ID,
			Version:    saved.Version,
			Snapshot:   saved.Redacted(),
			AuthorID:   actor.AuditActor().UserID,
		})
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

type conflictRequest struct {
	tenantInfo pagination.TenantInfo
	providerID pulid.ID
	loaded     int64
	cause      error
}

func (s *Service) explainConflict(ctx context.Context, req *conflictRequest) error {
	if !errortypes.IsVersionMismatchError(req.cause) {
		return req.cause
	}
	current, err := s.repo.GetByID(ctx, repositories.GetAIProviderByIDRequest{
		ID:         req.providerID,
		TenantInfo: req.tenantInfo,
	})
	if err != nil {
		return req.cause
	}

	return editconflict.Explain(ctx, s.versions, s.l, &editconflict.ExplainRequest[aiprovider.Provider]{
		TenantInfo:     req.tenantInfo,
		Kind:           settingversion.KindAIProvider,
		SubjectID:      current.ID,
		Loaded:         req.loaded,
		Current:        current.Redacted(),
		CurrentVersion: current.Version,
		UpdatedAt:      current.UpdatedAt,
		Rules:          aiprovider.ChangeRules,
		Cause:          req.cause,
	})
}
