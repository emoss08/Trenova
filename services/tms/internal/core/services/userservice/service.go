package userservice

import (
	"context"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	Repo              repositories.UserRepository
	RoleRepository    repositories.RoleRepository
	SessionRepository repositories.SessionRepository
	AuditService      services.AuditService
	Realtime          services.RealtimeService
	Storage           storage.Client
	Config            *config.Config
	Validator         *Validator
}

type Service struct {
	l            *zap.Logger
	repo         repositories.UserRepository
	roleRepo     repositories.RoleRepository
	sr           repositories.SessionRepository
	auditService services.AuditService
	realtime     services.RealtimeService
	storage      storage.Client
	storageCfg   *config.StorageConfig
	validator    *Validator
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.user"),
		sr:           p.SessionRepository,
		repo:         p.Repo,
		roleRepo:     p.RoleRepository,
		auditService: p.AuditService,
		realtime:     p.Realtime,
		storage:      p.Storage,
		storageCfg:   p.Config.GetStorageConfig(),
		validator:    p.Validator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListUsersRequest,
) (*pagination.ListResult[*tenant.User], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*tenant.User], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateUserStatusRequest,
) ([]*tenant.User, error) {
	log := s.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	originalEntities, err := s.repo.GetByIDs(ctx, repositories.GetUsersByIDsRequest{
		TenantInfo: req.TenantInfo,
		UserIDs:    req.UserIDs,
	})
	if err != nil {
		log.Error("failed to get original users", zap.Error(err))
		return nil, err
	}

	entities, err := s.repo.BulkUpdateStatus(ctx, req)
	if err != nil {
		log.Error("failed to bulk update user status", zap.Error(err))
		return nil, err
	}

	entries := auditservice.BuildBulkLogEntries(
		&auditservice.BulkLogEntriesParams[*tenant.User]{
			Resource:  permission.ResourceUser,
			Operation: permission.OpUpdate,
			UserID:    req.TenantInfo.UserID,
			Updated:   entities,
			Originals: originalEntities,
		},
		auditservice.WithComment("User status updated"),
	)

	if err = s.auditService.LogActions(entries); err != nil {
		log.Error("failed to log audit actions", zap.Error(err))
		return nil, err
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ActorUserID:    req.TenantInfo.UserID,
		Resource:       permission.ResourceUser.String(),
		Action:         "bulk_updated",
	}); err != nil {
		log.Warn("failed to publish user invalidation", zap.Error(err))
	}

	return entities, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetUserByIDRequest,
) (*tenant.User, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetOrganizations(
	ctx context.Context,
	userID pulid.ID,
	currentOrgID pulid.ID,
	currentBusinessUnitID pulid.ID,
) ([]*repositories.UserOrganizationResponse, error) {
	log := s.l.With(
		zap.String("operation", "GetUserOrganizations"),
		zap.String("userID", userID.String()),
	)

	results, err := s.repo.GetOrganizations(ctx, userID)
	if err != nil {
		log.Error("failed to get user organizations", zap.Error(err))
		return nil, err
	}

	isBusinessUnitAdmin := false
	if s.roleRepo != nil {
		isBusinessUnitAdmin, err = s.roleRepo.HasBusinessUnitAdminAccess(ctx, userID, currentOrgID)
		if err != nil {
			log.Error("failed to check business unit admin access", zap.Error(err))
			return nil, err
		}
	}

	if isBusinessUnitAdmin {
		orgsInBU, orgErr := s.repo.GetOrganizationsByBusinessUnit(ctx, currentBusinessUnitID)
		if orgErr != nil {
			log.Error("failed to get business unit organizations", zap.Error(orgErr))
			return nil, orgErr
		}

		membershipByOrgID := make(map[pulid.ID]*tenant.OrganizationMembership, len(results))
		for _, membership := range results {
			membershipByOrgID[membership.OrganizationID] = membership
		}

		orgs := make([]*repositories.UserOrganizationResponse, len(orgsInBU))
		for i, org := range orgsInBU {
			isDefault := false
			if membership, ok := membershipByOrgID[org.ID]; ok {
				isDefault = membership.IsDefault
			}

			orgs[i] = &repositories.UserOrganizationResponse{
				ID:        org.ID,
				Name:      org.Name,
				City:      org.City,
				State:     organizationStateName(org),
				LogoURL:   org.LogoURL,
				IsDefault: isDefault,
				IsCurrent: org.ID == currentOrgID,
			}
		}

		return orgs, nil
	}

	orgs := make([]*repositories.UserOrganizationResponse, len(results))
	for i, r := range results {
		if r == nil || r.Organization == nil {
			continue
		}

		orgs[i] = &repositories.UserOrganizationResponse{
			ID:        r.OrganizationID,
			Name:      r.Organization.Name,
			City:      r.Organization.City,
			State:     organizationStateName(r.Organization),
			LogoURL:   r.Organization.LogoURL,
			IsDefault: r.IsDefault,
			IsCurrent: r.OrganizationID == currentOrgID,
		}
	}

	return orgs, nil
}

func (s *Service) SwitchOrganization(
	ctx context.Context,
	req repositories.SwitchOrganizationRequest,
) (*tenant.User, error) {
	log := s.l.With(
		zap.String("operation", "SwitchOrganization"),
		zap.Any("request", req),
	)

	sess, err := s.sr.Get(ctx, req.SessionID)
	if err != nil {
		log.Error("failed to get session", zap.Error(err))
		return nil, errortypes.NewAuthenticationError("Invalid session")
	}

	if err = sess.Validate(); err != nil {
		log.Error("session validation failed", zap.Error(err))
		return nil, errortypes.NewAuthenticationError("Session expired")
	}

	orgs, err := s.repo.GetOrganizations(ctx, sess.UserID)
	if err != nil {
		log.Error("failed to get user organizations", zap.Error(err))
		return nil, err
	}

	var targetOrg *tenant.OrganizationMembership
	for _, membership := range orgs {
		if membership.OrganizationID == req.OrganizationID {
			targetOrg = membership
			break
		}
	}

	targetBusinessUnitID := pulid.ID("")
	if targetOrg != nil {
		targetBusinessUnitID = targetOrg.BusinessUnitID
	} else {
		isBUAdmin := false
		var buErr error
		if s.roleRepo != nil {
			isBUAdmin, buErr = s.roleRepo.HasBusinessUnitAdminAccess(ctx, sess.UserID, req.OrganizationID)
		}
		if buErr != nil {
			log.Error("failed to check business unit admin access", zap.Error(buErr))
			return nil, buErr
		}

		if !isBUAdmin {
			log.Warn("user attempted to switch to unauthorized organization")
			return nil, errortypes.NewAuthorizationError("You do not have access to this organization")
		}

		org, orgErr := s.repo.GetOrganizationByID(ctx, req.OrganizationID)
		if orgErr != nil {
			log.Error("failed to lookup target organization", zap.Error(orgErr))
			return nil, orgErr
		}
		targetBusinessUnitID = org.BusinessUnitID
	}

	if err = s.repo.UpdateCurrentOrganization(
		ctx,
		sess.UserID,
		req.OrganizationID,
		targetBusinessUnitID,
	); err != nil {
		log.Error("failed to update user's current organization", zap.Error(err))
		return nil, err
	}

	sess.OrganizationID = req.OrganizationID
	sess.BusinessUnitID = targetBusinessUnitID

	if err = s.sr.Update(ctx, sess); err != nil {
		log.Error("failed to update session", zap.Error(err))
		return nil, err
	}

	user, err := s.repo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			UserID: sess.UserID,
			OrgID:  req.OrganizationID,
			BuID:   targetBusinessUnitID,
		},
		IncludeMemberships: true,
	})
	if err != nil {
		log.Error("failed to get updated user", zap.Error(err))
		return nil, err
	}

	log.Info("user switched organization successfully",
		zap.String("newOrgID", req.OrganizationID.String()),
	)

	return user, nil
}

func organizationStateName(org *tenant.Organization) string {
	if org == nil || org.State == nil {
		return ""
	}

	return org.State.Name
}

func (s *Service) UploadProfilePicture(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fileHeader *multipart.FileHeader,
) (*tenant.User, error) {
	if multiErr := s.validateProfilePictureFile(fileHeader); multiErr != nil {
		return nil, multiErr
	}

	user, err := s.repo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	})
	if err != nil {
		return nil, err
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to process uploaded profile picture").WithInternal(err)
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	key := fileutils.GenerateStoragePath(
		tenantInfo.OrgID.String(),
		"user/profile-picture",
		fileHeader.Filename,
	)

	if _, err = s.storage.Upload(ctx, &storage.UploadParams{
		Key:         key,
		ContentType: contentType,
		Size:        fileHeader.Size,
		Body:        file,
		Metadata: map[string]string{
			"original_name": fileHeader.Filename,
			"resource_type": "user-profile-picture",
			"resource_id":   tenantInfo.UserID.String(),
		},
	}); err != nil {
		return nil, errortypes.NewDatabaseError("Failed to upload profile picture").WithInternal(err)
	}

	previousProfilePicture := user.ProfilePicURL
	previousThumbnail := user.ThumbnailURL
	user.ProfilePicURL = key
	user.ThumbnailURL = key

	updatedUser, err := s.Update(ctx, user, tenantInfo.UserID)
	if err != nil {
		_ = s.storage.Delete(ctx, key)
		return nil, err
	}

	s.deleteProfilePictureObject(ctx, previousProfilePicture, key)
	if previousThumbnail != previousProfilePicture {
		s.deleteProfilePictureObject(ctx, previousThumbnail, key)
	}

	return updatedUser, nil
}

func (s *Service) DeleteProfilePicture(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.User, error) {
	user, err := s.repo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	})
	if err != nil {
		return nil, err
	}

	previousProfilePicture := user.ProfilePicURL
	previousThumbnail := user.ThumbnailURL
	user.ProfilePicURL = ""
	user.ThumbnailURL = ""

	updatedUser, err := s.Update(ctx, user, tenantInfo.UserID)
	if err != nil {
		return nil, err
	}

	s.deleteProfilePictureObject(ctx, previousProfilePicture, "")
	if previousThumbnail != previousProfilePicture {
		s.deleteProfilePictureObject(ctx, previousThumbnail, "")
	}

	return updatedUser, nil
}

func (s *Service) GetProfilePictureURL(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	variant string,
) (string, error) {
	user, err := s.repo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:   tenantInfo,
		LookupUserID: userID,
	})
	if err != nil {
		return "", err
	}

	profilePictureKey := user.ProfilePicURL
	if variant == "thumbnail" && user.ThumbnailURL != "" {
		profilePictureKey = user.ThumbnailURL
	}
	if profilePictureKey == "" {
		profilePictureKey = user.ProfilePicURL
	}
	if profilePictureKey == "" {
		return "", errortypes.NewNotFoundError("Profile picture not found")
	}

	if fileutils.IsExternalURL(profilePictureKey) {
		return profilePictureKey, nil
	}

	url, err := s.storage.GetPresignedURL(ctx, &storage.PresignedURLParams{
		Key:    profilePictureKey,
		Expiry: s.storageCfg.GetPresignedURLExpiry(),
	})
	if err != nil {
		return "", errortypes.NewDatabaseError("Failed to generate profile picture URL").WithInternal(err)
	}

	return url, nil
}

func (s *Service) validateProfilePictureFile(file *multipart.FileHeader) *errortypes.MultiError {
	me := errortypes.NewMultiError()
	if file == nil {
		me.Add("file", errortypes.ErrRequired, "Profile picture file is required")
		return me
	}

	if file.Size == 0 {
		me.Add("file", errortypes.ErrRequired, "Profile picture file cannot be empty")
	}

	if file.Size > s.storageCfg.GetMaxFileSize() {
		me.Add("file", errortypes.ErrInvalidLength, "Profile picture file exceeds maximum allowed size")
	}

	if !fileutils.IsSupportedImageContentType(file.Header.Get("Content-Type")) {
		me.Add(
			"file",
			errortypes.ErrInvalidFormat,
			"Only image files are allowed for profile pictures",
		)
	}

	if !fileutils.HasSupportedImageExtension(file.Filename) {
		me.Add("file", errortypes.ErrInvalidFormat, "Unsupported profile picture file extension")
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) deleteProfilePictureObject(ctx context.Context, objectKey, currentKey string) {
	if objectKey == "" || objectKey == currentKey || fileutils.IsExternalURL(objectKey) {
		return
	}

	if err := s.storage.Delete(ctx, objectKey); err != nil {
		s.l.Warn(
			"failed to delete profile picture object",
			zap.String("objectKey", objectKey),
			zap.Error(err),
		)
	}
}

func (s *Service) UpdateMySettings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req UpdateMySettingsRequest,
) (*tenant.User, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	user, err := s.repo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	})
	if err != nil {
		return nil, err
	}

	user.Timezone = req.Timezone
	user.TimeFormat = req.TimeFormat

	return s.Update(ctx, user, tenantInfo.UserID)
}

func (s *Service) ChangeMyPassword(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req ChangeMyPasswordRequest,
) (*tenant.User, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	user, err := s.repo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if err = user.VerifyCredentials(req.CurrentPassword); err != nil {
		return nil, err
	}

	hashedPassword, err := user.GeneratePassword(req.NewPassword)
	if err != nil {
		return nil, err
	}

	original := *user
	if err = s.repo.UpdatePassword(ctx, repositories.UpdateUserPasswordRequest{
		UserID:             tenantInfo.UserID,
		OrganizationID:     tenantInfo.OrgID,
		BusinessUnitID:     tenantInfo.BuID,
		Password:           hashedPassword,
		MustChangePassword: false,
	}); err != nil {
		return nil, err
	}

	user.Password = hashedPassword
	user.MustChangePassword = false

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceUser,
		ResourceID:     user.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         tenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(user),
		PreviousState:  jsonutils.MustToJSON(&original),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}, auditservice.WithComment("User password updated")); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
		return nil, err
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    tenantInfo.UserID,
		Resource:       "users",
		Action:         "password_updated",
		RecordID:       tenantInfo.UserID,
	}); err != nil {
		s.l.Warn("failed to publish user invalidation", zap.Error(err))
	}

	updatedUser, err := s.repo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	})
	if err != nil {
		return nil, err
	}

	return updatedUser, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *tenant.User,
	userID pulid.ID,
) (*tenant.User, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			UserID: userID,
			OrgID:  entity.CurrentOrganizationID,
			BuID:   entity.BusinessUnitID,
		},
		IncludeMemberships: true,
	})
	if err != nil {
		log.Error("failed to get user", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update user", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceUser,
		ResourceID:     updatedEntity.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.CurrentOrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("User updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
		return nil, err
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: updatedEntity.CurrentOrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
		ActorUserID:    userID,
		Resource:       permission.ResourceUser.String(),
		Action:         "updated",
		RecordID:       updatedEntity.ID,
		Entity:         updatedEntity,
	}); err != nil {
		log.Warn("failed to publish user invalidation", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) ListOrganizationMemberships(
	ctx context.Context,
	userID, businessUnitID pulid.ID,
) ([]*tenant.OrganizationMembership, error) {
	return s.repo.ListOrganizationMemberships(ctx, userID, businessUnitID)
}

func (s *Service) ReplaceOrganizationMemberships(
	ctx context.Context,
	actorID, userID, organizationID, businessUnitID pulid.ID,
	organizationIDs []pulid.ID,
) ([]*tenant.OrganizationMembership, error) {
	memberships, err := s.repo.ReplaceOrganizationMemberships(
		ctx,
		repositories.ReplaceOrganizationMembershipsRequest{
			ActorID:         actorID,
			UserID:          userID,
			BusinessUnitID:  businessUnitID,
			OrganizationIDs: organizationIDs,
		},
	)
	if err != nil {
		return nil, err
	}

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: organizationID,
		BusinessUnitID: businessUnitID,
		ActorUserID:    actorID,
		Resource:       permission.ResourceUser.String(),
		Action:         "updated",
		RecordID:       userID,
	}); err != nil {
		s.l.Warn("failed to publish user invalidation", zap.Error(err))
	}

	return memberships, nil
}
