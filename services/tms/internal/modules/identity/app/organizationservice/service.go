//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package organizationservice

import (
	"bytes"
	"context"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/objectkey"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	DB            kernelports.Transactor
	Repo          tenantports.OrganizationRepository
	AuditService  auditports.AuditService
	Storage       kernelports.Client
	StorageConfig *storageconfig.Config
	Validator     *Validator
	ContentGate   contentscanports.Gate
	PlanFields    permissionports.PlanFieldKeeper
}

type service struct {
	l            *zap.Logger
	db           kernelports.Transactor
	repo         tenantports.OrganizationRepository
	auditService auditports.AuditService
	storage      kernelports.Client
	storageCfg   *storageconfig.Config
	v            *Validator
	contentGate  contentscanports.Gate
	planFields   permissionports.PlanFieldKeeper
}

func New(p Params) tenantports.OrganizationService {
	return &service{
		l:            p.Logger.Named("service.organization"),
		db:           p.DB,
		repo:         p.Repo,
		storage:      p.Storage,
		auditService: p.AuditService,
		storageCfg:   p.StorageConfig,
		v:            p.Validator,
		contentGate:  p.ContentGate,
		planFields:   p.PlanFields,
	}
}

func (s *service) GetByID(
	ctx context.Context,
	req tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *service) GetByIDs(
	ctx context.Context,
	req tenantports.GetOrganizationsByIDsRequest,
) ([]*tenant.Organization, error) {
	return s.repo.GetByIDs(ctx, tenantports.GetOrganizationsByIDsQuery(req))
}

func (s *service) SelectOptions(
	ctx context.Context,
	req *tenantports.SelectOrganizationOptionsRequest,
) (*pagination.ListResult[*tenant.Organization], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *service) Update(
	ctx context.Context,
	entity *tenant.Organization,
) (*tenant.Organization, error) {
	log := s.l.With(zap.String("operation", "Update"), zap.String("orgID", entity.ID.String()))

	if err := s.keepPlanFields(ctx, entity); err != nil {
		return nil, err
	}

	if err := s.v.ValidateUpdate(ctx, entity); err != nil {
		return nil, err
	}

	var updatedEntity *tenant.Organization
	if err := s.db.Transact(
		ctx,
		kernelports.TxOptions{},
		func(txCtx context.Context) error {
			if err := s.guardCapabilityDisable(txCtx, entity); err != nil {
				return err
			}

			stored, err := s.repo.GetByID(txCtx, tenantports.GetOrganizationByIDRequest{
				TenantInfo: pagination.TenantInfo{OrgID: entity.ID, BuID: entity.BusinessUnitID},
			})
			if err != nil {
				return err
			}
			entity.LogoURL = stored.LogoURL
			entity.IconURL = stored.IconURL

			updated, err := s.repo.Update(txCtx, entity)
			if err != nil {
				log.Error("failed to update organization", zap.Error(err))
				return mapOrganizationUniqueConstraint(err)
			}

			updatedEntity = updated

			return nil
		},
	); err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func mapOrganizationUniqueConstraint(err error) error {
	if !dberror.IsUniqueConstraintViolation(err) {
		return err
	}

	multiErr := errortypes.NewMultiError()

	switch dberror.ExtractConstraintName(err) {
	case "idx_organizations_name_business_unit":
		multiErr.Add(
			"name",
			errortypes.ErrDuplicate,
			"Organization with this name already exists in this business unit",
		)
	case "idx_organizations_scac_business_unit":
		multiErr.Add(
			"scacCode",
			errortypes.ErrDuplicate,
			"Organization with this SCAC code already exists in this business unit",
		)
	case "idx_organizations_dot_business_unit":
		multiErr.Add(
			"dotNumber",
			errortypes.ErrDuplicate,
			"Organization with this DOT number already exists in this business unit",
		)
	case "idx_organizations_login_slug":
		multiErr.Add(
			"loginSlug",
			errortypes.ErrDuplicate,
			"Organization with this login slug already exists",
		)
	default:
		return err
	}

	return multiErr
}

func (s *service) UploadBrandImage(
	ctx context.Context,
	req *tenantports.UploadBrandImageRequest,
	userID pulid.ID,
) (*tenant.Organization, error) {
	log := s.l.With(
		zap.String("operation", "UploadBrandImage"),
		zap.String("image", string(req.Image)),
		zap.String("userID", req.TenantInfo.UserID.String()),
	)

	if multiErr := s.validateBrandImageFile(req.File); multiErr != nil {
		return nil, multiErr
	}

	content, err := fileutils.ReadMultipart(req.File)
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to process uploaded image").
			WithInternal(err)
	}
	if _, err = contentscanports.Check(ctx, s.contentGate, &contentscanports.ScanRequest{
		TenantInfo: req.TenantInfo,
		Source:     contentscanports.SourceUpload,
		FileName:   req.File.Filename,
		Content:    content,
	}); err != nil {
		return nil, contentscanports.RefusalError("file", err)
	}

	contentType := req.File.Header.Get("Content-Type")
	key := objectkey.Organization.NewKey(
		req.OrganizationID,
		"organization/"+string(req.Image),
		req.File.Filename,
	)

	if _, err = s.storage.Upload(ctx, &kernelports.UploadParams{
		Key:         key,
		ContentType: contentType,
		Size:        int64(len(content)),
		Body:        bytes.NewReader(content),
		Metadata: map[string]string{
			"original_name": req.File.Filename,
			"resource_type": "organization-" + string(req.Image),
			"resource_id":   req.OrganizationID.String(),
		},
	}); err != nil {
		return nil, errortypes.NewDatabaseError("Failed to upload organization image").
			WithInternal(err)
	}

	entity, err := s.repo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		_ = s.storage.Delete(ctx, key)
		return nil, err
	}

	original := *entity
	previousKey := entity.BrandImageKey(req.Image)
	entity.SetBrandImageKey(req.Image, key)

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		_ = s.storage.Delete(ctx, key)
		return nil, err
	}

	if previousKey != key {
		s.deleteBrandImageObject(ctx, req.OrganizationID, req.Image, previousKey)
	}

	if err = s.auditService.LogAction(ctx, &auditports.LogActionParams{
		Resource:       permission.ResourceOrganization,
		ResourceID:     updatedEntity.GetResourceID(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(&original),
		OrganizationID: entity.ID,
		BusinessUnitID: entity.BusinessUnitID,
	}, auditservice.WithComment("Organization updated")); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *service) GetBranding(
	ctx context.Context,
	req tenantports.GetBrandingRequest,
) (*tenantports.Branding, error) {
	entity, err := s.repo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, err
	}

	logoURL, err := s.brandImageURL(ctx, entity.LogoURL)
	if err != nil {
		return nil, err
	}
	iconURL, err := s.brandImageURL(ctx, entity.IconURL)
	if err != nil {
		return nil, err
	}

	return &tenantports.Branding{LogoURL: logoURL, IconURL: iconURL}, nil
}

func (s *service) brandImageURL(ctx context.Context, key string) (string, error) {
	if key == "" || fileutils.IsExternalURL(key) {
		return key, nil
	}

	url, err := s.storage.GetPresignedURL(ctx, &kernelports.PresignedURLParams{
		Key:    key,
		Expiry: s.storageCfg.GetPresignedURLExpiry(),
	})
	if err != nil {
		return "", errortypes.NewDatabaseError("Failed to generate organization image URL").
			WithInternal(err)
	}

	return url, nil
}

func (s *service) DeleteBrandImage(
	ctx context.Context,
	req tenantports.DeleteBrandImageRequest,
) (*tenant.Organization, error) {
	org, err := s.repo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, err
	}

	previousKey := org.BrandImageKey(req.Image)
	if previousKey == "" {
		return org, nil
	}

	updatedOrg, err := s.repo.ClearBrandImage(ctx, req.OrganizationID, req.Image, org.Version)
	if err != nil {
		return nil, err
	}

	s.deleteBrandImageObject(ctx, req.OrganizationID, req.Image, previousKey)

	if err = s.auditService.LogAction(ctx, &auditports.LogActionParams{
		Resource:       permission.ResourceOrganization,
		ResourceID:     updatedOrg.GetResourceID(),
		Operation:      permission.OpUpdate,
		UserID:         req.TenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(updatedOrg),
		PreviousState:  jsonutils.MustToJSON(org),
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
	}, auditservice.WithComment("Organization updated")); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	return updatedOrg, nil
}

func (s *service) deleteBrandImageObject(
	ctx context.Context,
	orgID pulid.ID,
	image tenant.BrandImage,
	key string,
) {
	if key == "" || fileutils.IsExternalURL(key) {
		return
	}

	if err := s.storage.Delete(ctx, key); err != nil {
		s.l.Warn(
			"failed to delete organization image object",
			zap.String("orgID", orgID.String()),
			zap.String("image", string(image)),
			zap.String("key", key),
			zap.Error(err),
		)
	}
}

func (s *service) validateBrandImageFile(file *multipart.FileHeader) error {
	multiErr := errortypes.NewMultiError()
	if file == nil {
		multiErr.Add("file", errortypes.ErrRequired, "Image file is required")
		return multiErr.Err()
	}

	if file.Size == 0 {
		multiErr.Add("file", errortypes.ErrRequired, "Image file cannot be empty")
	}

	if file.Size > s.storageCfg.GetMaxFileSize() {
		multiErr.Add("file", errortypes.ErrInvalidLength, "Image file exceeds maximum allowed size")
	}

	if !fileutils.IsSupportedImageContentType(file.Header.Get("Content-Type")) {
		multiErr.Add(
			"file",
			errortypes.ErrInvalidFormat,
			"Only image files are allowed for organization branding",
		)
	}

	if !fileutils.HasSupportedImageExtension(file.Filename) {
		multiErr.Add("file", errortypes.ErrInvalidFormat, "Unsupported image file extension")
	}

	return multiErr.Err()
}

// keepPlanFields keeps an update from changing the organization's settings
// that belong to an add-on its plan leaves out, such as whether brokerage is
// on without Carrier & Brokerage, before the update is validated.
func (s *service) keepPlanFields(ctx context.Context, entity *tenant.Organization) error {
	stored, err := s.repo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{OrgID: entity.ID, BuID: entity.BusinessUnitID},
	})
	if err != nil {
		return err
	}

	return s.planFields.Keep(ctx, permission.ResourceOrganization, entity, stored)
}
