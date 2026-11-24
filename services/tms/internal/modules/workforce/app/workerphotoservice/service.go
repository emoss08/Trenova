package workerphotoservice

import (
	"context"
	"mime/multipart"
	"slices"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/personphoto"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	photoFolder       = "worker/photo"
	photoResourceType = "worker-photo"
	photoField        = "photo"
)

type Repository interface {
	GetByID(ctx context.Context, req workerports.GetWorkerByIDRequest) (*worker.Worker, error)
	UpdatePhoto(
		ctx context.Context,
		req *workerports.UpdateWorkerPhotoRequest,
	) (*worker.Worker, error)
	GetPhotoKeys(
		ctx context.Context,
		req workerports.GetWorkersByIDsRequest,
	) ([]workerports.WorkerPhotoKeys, error)
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         workerports.WorkerRepository
	Photos       kernelports.Photos
	AuditService auditports.AuditService
	Realtime     realtimeports.RealtimeService
	DB           kernelports.Transactor
	Permissions  permissionports.PermissionEngine
	Registry     *permissioncatalog.Registry
}

type Service struct {
	l            *zap.Logger
	repo         Repository
	photos       kernelports.Photos
	permissions  permissionports.PermissionEngine
	registry     *permissioncatalog.Registry
	auditService auditports.AuditService
	realtime     realtimeports.RealtimeService
	db           kernelports.Transactor
}

//nolint:gocritic // fx passes params by value
func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.worker-photo"),
		repo:         p.Repo,
		photos:       p.Photos,
		permissions:  p.Permissions,
		registry:     p.Registry,
		auditService: p.AuditService,
		realtime:     p.Realtime,
		db:           p.DB,
	}
}

type UploadRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	Version    int64
	File       *multipart.FileHeader
	Actor      *permissionports.RequestActor
}

type RemoveRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	Version    int64
	Actor      *permissionports.RequestActor
}

type StageRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	File       *multipart.FileHeader
}

type ApplyRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	Photo      kernelports.StoredPhoto
	Actor      *permissionports.RequestActor
}

type Result struct {
	Photo   *personphoto.URLs
	Version int64
}

type change struct {
	tenantInfo pagination.TenantInfo
	original   *worker.Worker
	version    int64
	photo      kernelports.StoredPhoto
	actor      *permissionports.RequestActor
	comment    string
}

func (s *Service) Upload(ctx context.Context, req *UploadRequest) (*Result, error) {
	original, err := s.load(ctx, req.TenantInfo, req.WorkerID, req.Version)
	if err != nil {
		return nil, err
	}

	stored, err := s.Stage(ctx, &StageRequest{
		TenantInfo: req.TenantInfo,
		WorkerID:   req.WorkerID,
		File:       req.File,
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.write(ctx, &change{
		tenantInfo: req.TenantInfo,
		original:   original,
		version:    req.Version,
		photo:      *stored,
		actor:      req.Actor,
		comment:    "Photo updated",
	})
	if err != nil {
		s.Discard(
			ctx,
			WorkerRef{OrganizationID: req.TenantInfo.OrgID, WorkerID: req.WorkerID},
			stored,
		)
		return nil, err
	}

	return s.result(ctx, updated)
}

func (s *Service) Remove(ctx context.Context, req *RemoveRequest) (*Result, error) {
	original, err := s.load(ctx, req.TenantInfo, req.WorkerID, req.Version)
	if err != nil {
		return nil, err
	}
	if !original.HasPhoto() {
		return &Result{Version: original.Version}, nil
	}

	updated, err := s.write(ctx, &change{
		tenantInfo: req.TenantInfo,
		original:   original,
		version:    req.Version,
		actor:      req.Actor,
		comment:    "Photo removed",
	})
	if err != nil {
		return nil, err
	}

	return &Result{Version: updated.Version}, nil
}

func (s *Service) Stage(
	ctx context.Context,
	req *StageRequest,
) (*kernelports.StoredPhoto, error) {
	return s.photos.Store(ctx, &kernelports.StorePhotoRequest{
		TenantInfo:   req.TenantInfo,
		File:         req.File,
		Holder:       holder(req.TenantInfo.OrgID, req.WorkerID),
		ResourceType: photoResourceType,
	})
}

func (s *Service) Apply(ctx context.Context, req *ApplyRequest) (*worker.Worker, error) {
	original, err := s.repo.GetByID(ctx, workerports.GetWorkerByIDRequest{
		ID:         req.WorkerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return s.write(ctx, &change{
		tenantInfo: req.TenantInfo,
		original:   original,
		version:    original.Version,
		photo:      req.Photo,
		actor:      req.Actor,
		comment:    "Photo updated from the driver's request",
	})
}

type WorkerRef struct {
	OrganizationID pulid.ID
	WorkerID       pulid.ID
}

func holder(organizationID, workerID pulid.ID) kernelports.PhotoHolder {
	return kernelports.PhotoHolder{
		OrganizationID: organizationID,
		Folder:         photoFolder,
		ID:             workerID,
	}
}

func (s *Service) Discard(ctx context.Context, ref WorkerRef, photo *kernelports.StoredPhoto) {
	if photo == nil {
		return
	}
	s.photos.Discard(ctx, &kernelports.PhotoKeys{
		Holder:       holder(ref.OrganizationID, ref.WorkerID),
		Key:          photo.Key,
		ThumbnailKey: photo.ThumbnailKey,
	})
}

func (s *Service) URLs(ctx context.Context, entity *worker.Worker) (*personphoto.URLs, error) {
	if entity == nil {
		return nil, nil //nolint:nilnil // no worker has no photo
	}
	return s.photos.Sign(ctx, &kernelports.PhotoKeys{
		Holder:       holder(entity.OrganizationID, entity.ID),
		Key:          entity.PhotoKey,
		ThumbnailKey: entity.PhotoThumbnailKey,
	})
}

func (s *Service) ProposedURLs(
	ctx context.Context,
	ref WorkerRef,
	photo *kernelports.StoredPhoto,
) (*personphoto.URLs, error) {
	if photo == nil {
		return nil, nil //nolint:nilnil // nothing staged has no photo
	}
	return s.photos.Sign(ctx, &kernelports.PhotoKeys{
		Holder:       holder(ref.OrganizationID, ref.WorkerID),
		Key:          photo.Key,
		ThumbnailKey: photo.ThumbnailKey,
	})
}

func (s *Service) URLsByWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerIDs []pulid.ID,
) (map[pulid.ID]*personphoto.URLs, error) {
	ids := sliceutils.Dedupe(slices.DeleteFunc(slices.Clone(workerIDs), pulid.ID.IsNil))
	if len(ids) == 0 {
		return map[pulid.ID]*personphoto.URLs{}, nil
	}
	visible, err := s.photosVisible(ctx, tenantInfo)
	if err != nil || !visible {
		return map[pulid.ID]*personphoto.URLs{}, err
	}

	rows, err := s.repo.GetPhotoKeys(ctx, workerports.GetWorkersByIDsRequest{
		TenantInfo: tenantInfo,
		WorkerIDs:  ids,
	})
	if err != nil {
		return nil, err
	}

	out := make(map[pulid.ID]*personphoto.URLs, len(rows))
	for _, row := range rows {
		urls, signErr := s.photos.Sign(ctx, &kernelports.PhotoKeys{
			Holder:       holder(tenantInfo.OrgID, row.WorkerID),
			Key:          row.PhotoKey,
			ThumbnailKey: row.PhotoThumbnailKey,
		})
		if signErr != nil {
			return nil, signErr
		}
		if urls != nil {
			out[row.WorkerID] = urls
		}
	}
	return out, nil
}

func (s *Service) photosVisible(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	if tenantInfo.UserID.IsNil() {
		return false, nil
	}
	detail, err := s.permissions.GetResourcePermissions(
		ctx,
		tenantInfo.UserID,
		tenantInfo.OrgID,
		permission.ResourceWorker.String(),
	)
	if err != nil {
		return false, err
	}
	if detail == nil || !slices.Contains(detail.Operations, permission.OpRead) {
		return false, nil
	}
	return detail.MaxSensitivity.CanAccess(
		s.registry.GetFieldSensitivity(permission.ResourceWorker.String(), photoField),
	), nil
}

func (s *Service) load(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	version int64,
) (*worker.Worker, error) {
	original, err := s.repo.GetByID(ctx, workerports.GetWorkerByIDRequest{
		ID:         workerID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.Version != version {
		return nil, dberror.CreateVersionMismatchError("Worker", workerID.String())
	}
	return original, nil
}

func (s *Service) write(ctx context.Context, c *change) (*worker.Worker, error) {
	previous := kernelports.StoredPhoto{
		Key:          c.original.PhotoKey,
		ThumbnailKey: c.original.PhotoThumbnailKey,
	}
	auditActor := c.actor.AuditActor()

	var updated *worker.Worker
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var txErr error
		updated, txErr = s.repo.UpdatePhoto(txCtx, &workerports.UpdateWorkerPhotoRequest{
			TenantInfo:        c.tenantInfo,
			WorkerID:          c.original.ID,
			Version:           c.version,
			PhotoKey:          c.photo.Key,
			PhotoThumbnailKey: c.photo.ThumbnailKey,
		})
		if txErr != nil {
			return txErr
		}

		if txErr = s.auditService.LogAction(txCtx, &auditports.LogActionParams{
			Resource:       permission.ResourceWorker,
			ResourceID:     updated.GetResourceID(),
			Operation:      permission.OpUpdate,
			UserID:         auditActor.UserID,
			PrincipalType:  auditActor.PrincipalType,
			PrincipalID:    auditActor.PrincipalID,
			APIKeyID:       auditActor.APIKeyID,
			CurrentState:   jsonutils.MustToJSON(updated),
			PreviousState:  jsonutils.MustToJSON(c.original),
			OrganizationID: updated.GetOrganizationID(),
			BusinessUnitID: updated.GetBusinessUnitID(),
		},
			auditservice.WithComment(c.comment),
			auditservice.WithMetadata(map[string]any{
				"photoChange": photoChange(previous.Key, c.photo.Key),
			}),
			auditservice.WithSubjectOf(updated),
		); txErr != nil {
			return txErr
		}

		invalidation := &realtimeinvalidation.PublishParams{
			OrganizationID: updated.GetOrganizationID(),
			BusinessUnitID: updated.GetBusinessUnitID(),
			ActorUserID:    auditActor.UserID,
			ActorType:      auditActor.PrincipalType,
			ActorID:        auditActor.PrincipalID,
			ActorAPIKeyID:  auditActor.APIKeyID,
			Resource:       permission.ResourceWorker.String(),
			Action:         "updated",
			RecordID:       updated.GetID(),
		}
		kernelports.AfterCommit(txCtx, func(committed context.Context) {
			if previous.Key != c.photo.Key {
				s.Discard(
					committed,
					WorkerRef{OrganizationID: c.tenantInfo.OrgID, WorkerID: c.original.ID},
					&previous,
				)
			}
			if publishErr := realtimeinvalidation.Publish(
				committed,
				s.realtime,
				invalidation,
			); publishErr != nil {
				s.l.Warn("failed to publish worker invalidation", zap.Error(publishErr))
			}
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) result(ctx context.Context, updated *worker.Worker) (*Result, error) {
	urls, err := s.URLs(ctx, updated)
	if err != nil {
		return nil, err
	}
	return &Result{Photo: urls, Version: updated.Version}, nil
}

func photoChange(previousKey, nextKey string) string {
	switch {
	case nextKey == "":
		return "removed"
	case previousKey == "":
		return "added"
	default:
		return "replaced"
	}
}
