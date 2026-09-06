package workercredentialservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) ListTypes(
	ctx context.Context,
	req *repositories.ListCredentialTypesRequest,
) (*pagination.CursorListResult[*worker.WorkerCredentialType], error) {
	if req.Filter != nil {
		if err := s.ensureSystemTypes(ctx, req.Filter.TenantInfo); err != nil {
			return nil, err
		}
	}
	return s.repo.ListTypes(ctx, req)
}

func (s *Service) GetType(
	ctx context.Context,
	req *repositories.GetCredentialTypeByIDRequest,
) (*worker.WorkerCredentialType, error) {
	return s.repo.GetTypeByID(ctx, req)
}

// ActiveTypes returns the catalog a worker can be measured against, seeding the
// system rows first so an organisation created after the registry migration
// still starts with the FMCSA set.
func (s *Service) ActiveTypes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.WorkerCredentialType, error) {
	if err := s.ensureSystemTypes(ctx, tenantInfo); err != nil {
		return nil, err
	}
	return s.repo.ListActiveTypes(ctx, tenantInfo)
}

func (s *Service) ensureSystemTypes(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return nil
	}
	key := tenantInfo.OrgID.String() + ":" + tenantInfo.BuID.String()
	if _, seeded := s.seededTenants.Load(key); seeded {
		return nil
	}
	inserted, err := s.repo.EnsureSystemTypes(ctx, tenantInfo, worker.SystemCredentialTypes())
	if err != nil {
		return err
	}
	if inserted > 0 {
		s.l.Info("seeded system credential types",
			zap.String("orgId", tenantInfo.OrgID.String()),
			zap.Int("inserted", inserted))
	}
	s.seededTenants.Store(key, struct{}{})
	return nil
}

func (s *Service) validateType(ctx context.Context, entity *worker.WorkerCredentialType) error {
	entity.NormalizeCode()
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if entity.Code != "" {
		exists, err := s.repo.TypeCodeExists(ctx, &repositories.CredentialTypeCodeExistsRequest{
			TenantInfo: typeTenant(entity),
			Code:       entity.Code,
			ExcludeID:  entity.ID,
		})
		if err != nil {
			return err
		}
		if exists {
			multiErr.Add(
				"code",
				errortypes.ErrDuplicate,
				"A credential type with this code already exists",
			)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) CreateType(
	ctx context.Context,
	entity *worker.WorkerCredentialType,
	userID pulid.ID,
) (*worker.WorkerCredentialType, error) {
	log := s.l.With(zap.String("operation", "CreateType"))

	entity.IsSystem = false
	entity.ProfileField = worker.CredentialProfileFieldNone
	if err := s.validateType(ctx, entity); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateType(ctx, entity)
	if err != nil {
		log.Error("failed to create credential type", zap.Error(err))
		return nil, err
	}

	s.auditType(created, nil, permission.OpCreate, userID, "Credential type created", log)
	s.publish(
		ctx,
		typeTenant(created),
		realtimeResourceType,
		permission.OpCreate,
		created.ID,
		userID,
	)

	return created, nil
}

func (s *Service) UpdateType(
	ctx context.Context,
	entity *worker.WorkerCredentialType,
	userID pulid.ID,
) (*worker.WorkerCredentialType, error) {
	log := s.l.With(zap.String("operation", "UpdateType"), zap.String("id", entity.ID.String()))

	original, err := s.repo.GetTypeByID(ctx, &repositories.GetCredentialTypeByIDRequest{
		ID:         entity.ID,
		TenantInfo: typeTenant(entity),
	})
	if err != nil {
		return nil, err
	}

	entity.IsSystem = original.IsSystem
	entity.ProfileField = original.ProfileField
	if original.IsSystem {
		entity.Code = original.Code
	}
	if err = s.validateType(ctx, entity); err != nil {
		return nil, err
	}
	if original.Status == domaintypes.StatusActive && entity.Status != domaintypes.StatusActive {
		if err = s.requireTypeRetirable(ctx, original); err != nil {
			return nil, err
		}
	}

	updated, err := s.repo.UpdateType(ctx, entity)
	if err != nil {
		log.Error("failed to update credential type", zap.Error(err))
		return nil, err
	}

	s.auditType(updated, original, permission.OpUpdate, userID, "Credential type updated", log)
	s.publish(
		ctx,
		typeTenant(updated),
		realtimeResourceType,
		permission.OpUpdate,
		updated.ID,
		userID,
	)

	return updated, nil
}

type TypeStatusChangeRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Version    int64
	UserID     pulid.ID
}

func (s *Service) ArchiveType(
	ctx context.Context,
	req *TypeStatusChangeRequest,
) (*worker.WorkerCredentialType, error) {
	return s.changeTypeStatus(ctx, req, domaintypes.StatusInactive, permission.OpArchive)
}

func (s *Service) RestoreType(
	ctx context.Context,
	req *TypeStatusChangeRequest,
) (*worker.WorkerCredentialType, error) {
	return s.changeTypeStatus(ctx, req, domaintypes.StatusActive, permission.OpRestore)
}

func (s *Service) changeTypeStatus(
	ctx context.Context,
	req *TypeStatusChangeRequest,
	target domaintypes.Status,
	operation permission.Operation,
) (*worker.WorkerCredentialType, error) {
	log := s.l.With(zap.String("operation", string(operation)), zap.String("id", req.ID.String()))

	original, err := s.repo.GetTypeByID(ctx, &repositories.GetCredentialTypeByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.Status == target {
		return original, nil
	}
	if req.Version > 0 && original.Version != req.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Credential type was changed by someone else. Reload and try again",
		)
	}
	if target == domaintypes.StatusInactive {
		if err = s.requireTypeRetirable(ctx, original); err != nil {
			return nil, err
		}
	}

	updated := *original
	updated.Status = target
	saved, err := s.repo.UpdateType(ctx, &updated)
	if err != nil {
		log.Error("failed to change credential type status", zap.Error(err))
		return nil, err
	}

	s.auditType(
		saved,
		original,
		operation,
		req.UserID,
		fmt.Sprintf("Credential type %s", target),
		log,
	)
	s.publish(ctx, typeTenant(saved), realtimeResourceType, operation, saved.ID, req.UserID)

	return saved, nil
}

// requireTypeRetirable keeps mirrored system types (they back a profile column
// and the dispatch rules read that column) and any type still held by a worker
// from being switched off.
func (s *Service) requireTypeRetirable(
	ctx context.Context,
	entity *worker.WorkerCredentialType,
) error {
	if entity.ProfileField.IsSet() {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"This credential type mirrors the worker profile and cannot be deactivated",
		)
	}
	count, err := s.repo.CountCredentialsByType(ctx, &repositories.CountCredentialsByTypeRequest{
		TenantInfo: typeTenant(entity),
		TypeID:     entity.ID,
		ActiveOnly: true,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			fmt.Sprintf(
				"%d worker%s still hold%s this credential. Archive those first",
				count,
				plural(count, "", "s"),
				plural(count, "s", ""),
			),
		)
	}
	return nil
}

func (s *Service) CountActiveCredentials(
	ctx context.Context,
	entity *worker.WorkerCredentialType,
) (int, error) {
	return s.repo.CountCredentialsByType(ctx, &repositories.CountCredentialsByTypeRequest{
		TenantInfo: typeTenant(entity),
		TypeID:     entity.ID,
		ActiveOnly: true,
	})
}

func (s *Service) CountActiveCredentialsByTypes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typeIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	return s.repo.CountCredentialsByTypeIDs(ctx, &repositories.CountCredentialsByTypeIDsRequest{
		TenantInfo: tenantInfo,
		TypeIDs:    typeIDs,
		ActiveOnly: true,
	})
}

func plural(count int, one, many string) string {
	if count == 1 {
		return one
	}
	return many
}

func typeTenant(entity *worker.WorkerCredentialType) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}
