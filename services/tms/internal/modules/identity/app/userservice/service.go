package userservice

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"slices"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/domaintypes"
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
	Repo              tenantports.UserRepository
	RoleRepository    permissionports.RoleRepository
	SessionRepository sessionports.SessionRepository
	AuditService      auditports.AuditService
	Auditor           permissionports.SecurityAuditor
	Realtime          realtimeports.RealtimeService
	Storage           kernelports.Client
	Config            *config.Config
	Validator         *Validator
	DB                kernelports.Transactor
	Roles             RoleAssigner
	Invites           AccountInviter
	PermissionEngine  permissionports.PermissionEngine
	AuthEventHistory  iamports.AuthEventHistoryRepository `optional:"true"`
	AuthEvents        iamports.AuthEventRecorder          `optional:"true"`
	ContentGate       contentscanports.Gate
}

type Service struct {
	l            *zap.Logger
	repo         tenantports.UserRepository
	roleRepo     permissionports.RoleRepository
	sr           sessionports.SessionRepository
	auditService auditports.AuditService
	auditor      permissionports.SecurityAuditor
	realtime     realtimeports.RealtimeService
	storage      kernelports.Client
	storageCfg   *config.StorageConfig
	validator    *Validator
	db           kernelports.Transactor
	roles        RoleAssigner
	invites      AccountInviter
	permissions  permissionports.PermissionEngine
	signIns      iamports.AuthEventHistoryRepository
	authEvents   iamports.AuthEventRecorder
	contentGate  contentscanports.Gate
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:            p.Logger.Named("service.user"),
		sr:           p.SessionRepository,
		repo:         p.Repo,
		roleRepo:     p.RoleRepository,
		auditService: p.AuditService,
		auditor:      p.Auditor,
		realtime:     p.Realtime,
		storage:      p.Storage,
		storageCfg:   p.Config.GetStorageConfig(),
		validator:    p.Validator,
		db:           p.DB,
		roles:        p.Roles,
		invites:      p.Invites,
		permissions:  p.PermissionEngine,
		signIns:      p.AuthEventHistory,
		authEvents:   p.AuthEvents,
		contentGate:  p.ContentGate,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *tenantports.ListUsersRequest,
) (*pagination.ListResult[*tenant.User], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *tenantports.ListUserConnectionRequest,
) (*pagination.CursorListResult[*tenant.User], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*tenant.User], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *tenantports.BulkUpdateUserStatusRequest,
) ([]*tenant.User, error) {
	log := s.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	if req.Status != domaintypes.StatusActive &&
		slices.Contains(req.UserIDs, req.TenantInfo.UserID) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrForbidden,
			"You cannot deactivate your own account",
		)
	}

	originalEntities, err := s.repo.GetByIDs(ctx, tenantports.GetUsersByIDsRequest{
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

	if err = s.endSessionsAfterBulkStatus(
		ctx,
		req.TenantInfo,
		originalEntities,
		entities,
	); err != nil {
		return nil, err
	}

	return entities, nil
}

func (s *Service) endSessionsAfterBulkStatus(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	originals, updated []*tenant.User,
) error {
	before := make(map[pulid.ID]*tenant.User, len(originals))
	for _, user := range originals {
		before[user.ID] = user
	}

	var errs []error
	for _, user := range updated {
		original, ok := before[user.ID]
		if !ok {
			original = &tenant.User{Status: domaintypes.StatusActive}
		}

		var err error
		if lostSignInAccess(original, user) {
			err = s.endAllSessions(ctx, tenantInfo, user.ID)
		} else {
			err = s.endLingeringSessions(ctx, tenantInfo, user)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (s *Service) GetByID(
	ctx context.Context,
	req tenantports.GetUserByIDRequest,
) (*tenant.User, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetOrganizations(
	ctx context.Context,
	userID pulid.ID,
	currentOrgID pulid.ID,
) ([]*tenantports.UserOrganizationResponse, error) {
	log := s.l.With(
		zap.String("operation", "GetUserOrganizations"),
		zap.String("userID", userID.String()),
	)

	results, err := s.repo.GetOrganizations(ctx, userID)
	if err != nil {
		log.Error("failed to get user organizations", zap.Error(err))
		return nil, err
	}

	orgs := make([]*tenantports.UserOrganizationResponse, len(results))
	for i, r := range results {
		if r == nil || r.Organization == nil {
			continue
		}

		orgs[i] = &tenantports.UserOrganizationResponse{
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
	req tenantports.SwitchOrganizationRequest,
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

	var targetBusinessUnitID pulid.ID
	if targetOrg != nil {
		targetBusinessUnitID = targetOrg.BusinessUnitID
	} else {
		log.Warn("user attempted to switch to unauthorized organization")
		return nil, errortypes.NewAuthorizationError(
			"You do not have access to this organization",
		)
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
	sess.ActiveRoleIDs = nil

	if err = s.sr.Update(ctx, sess); err != nil {
		log.Error("failed to update session", zap.Error(err))
		return nil, err
	}

	user, err := s.repo.GetByID(ctx, tenantports.GetUserByIDRequest{
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

	user, err := s.repo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	})
	if err != nil {
		return nil, err
	}

	content, err := fileutils.ReadMultipart(fileHeader)
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to process uploaded profile picture").
			WithInternal(err)
	}
	if _, err = contentscanports.Check(ctx, s.contentGate, &contentscanports.ScanRequest{
		TenantInfo: tenantInfo,
		Source:     contentscanports.SourceUpload,
		FileName:   fileHeader.Filename,
		Content:    content,
	}); err != nil {
		return nil, contentscanports.RefusalError("file", err)
	}

	contentType := fileHeader.Header.Get("Content-Type")
	key := fileutils.GenerateStoragePath(
		tenantInfo.OrgID.String(),
		"user/profile-picture",
		fileHeader.Filename,
	)

	if _, err = s.storage.Upload(ctx, &kernelports.UploadParams{
		Key:         key,
		ContentType: contentType,
		Size:        int64(len(content)),
		Body:        bytes.NewReader(content),
		Metadata: map[string]string{
			"original_name": fileHeader.Filename,
			"resource_type": "user-profile-picture",
			"resource_id":   tenantInfo.UserID.String(),
		},
	}); err != nil {
		return nil, errortypes.NewDatabaseError("Failed to upload profile picture").
			WithInternal(err)
	}

	original := *user
	previousProfilePicture := user.ProfilePicURL
	previousThumbnail := user.ThumbnailURL
	user.ProfilePicURL = key
	user.ThumbnailURL = key

	cols := tenantdb.Users
	updatedUser, err := s.updateColumns(ctx, &columnUpdate{
		tenantInfo: tenantInfo,
		user:       user,
		original:   &original,
		columns:    []string{columnName(cols.ProfilePicURL.Col), columnName(cols.ThumbnailURL.Col)},
		comment:    "Profile picture updated",
	})
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
	user, err := s.repo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	})
	if err != nil {
		return nil, err
	}

	original := *user
	previousProfilePicture := user.ProfilePicURL
	previousThumbnail := user.ThumbnailURL
	user.ProfilePicURL = ""
	user.ThumbnailURL = ""

	cols := tenantdb.Users
	updatedUser, err := s.updateColumns(ctx, &columnUpdate{
		tenantInfo: tenantInfo,
		user:       user,
		original:   &original,
		columns:    []string{columnName(cols.ProfilePicURL.Col), columnName(cols.ThumbnailURL.Col)},
		comment:    "Profile picture removed",
	})
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
	user, err := s.repo.GetByID(ctx, tenantports.GetUserByIDRequest{
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

	url, err := s.storage.GetPresignedURL(ctx, &kernelports.PresignedURLParams{
		Key:    profilePictureKey,
		Expiry: s.storageCfg.GetPresignedURLExpiry(),
	})
	if err != nil {
		return "", errortypes.NewDatabaseError("Failed to generate profile picture URL").
			WithInternal(err)
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
		me.Add(
			"file",
			errortypes.ErrInvalidLength,
			"Profile picture file exceeds maximum allowed size",
		)
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

	user, err := s.repo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	})
	if err != nil {
		return nil, err
	}

	original := *user
	user.Timezone = req.Timezone
	user.TimeFormat = req.TimeFormat

	cols := tenantdb.Users
	columns := []string{columnName(cols.Timezone.Col), columnName(cols.TimeFormat.Col)}
	if req.Locale != "" {
		user.Locale = req.Locale
		columns = append(columns, columnName(cols.Locale.Col))
	}

	updated, err := s.updateColumns(ctx, &columnUpdate{
		tenantInfo: tenantInfo,
		user:       user,
		original:   &original,
		columns:    columns,
		comment:    "Settings updated",
	})
	if err != nil {
		return nil, err
	}

	if req.Locale != "" {
		s.refreshSessionLocale(ctx, req.SessionID, tenantInfo.UserID, req.Locale)
	}

	return updated, nil
}

func (s *Service) refreshSessionLocale(
	ctx context.Context,
	sessionID, userID pulid.ID,
	locale string,
) {
	if sessionID.IsNil() {
		return
	}

	log := s.l.With(
		zap.String("operation", "refreshSessionLocale"),
		zap.String("sessionID", sessionID.String()),
	)

	sess, err := s.sr.Get(ctx, sessionID)
	if err != nil {
		log.Warn("failed to load session for locale refresh", zap.Error(err))
		return
	}
	if sess.UserID != userID || sess.Locale == locale {
		return
	}

	sess.Locale = locale
	if err = s.sr.Update(ctx, sess); err != nil {
		log.Warn("failed to refresh session locale", zap.Error(err))
	}
}

func (s *Service) ChangeMyPassword(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req ChangeMyPasswordRequest,
) (*tenant.User, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	user, err := s.repo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if err = tenant.CheckNewPassword(
		"newPassword",
		req.NewPassword,
		user.EmailAddress,
	); err != nil {
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
	if err = s.repo.UpdatePassword(ctx, tenantports.UpdateUserPasswordRequest{
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

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceUser,
		ResourceID:     user.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         tenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(user),
		PreviousState:  jsonutils.MustToJSON(&original),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Critical:       true,
	},
		auditservice.WithComment("User password updated"),
		auditservice.WithCategory(audit.CategoryUser),
		auditservice.WithRequest(ctx),
	); err != nil {
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

	updatedUser, err := s.repo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:         tenantInfo,
		IncludeMemberships: true,
	})
	if err != nil {
		return nil, err
	}

	return updatedUser, nil
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
	if len(organizationIDs) == 0 {
		return nil, tenant.NewNoOrganizationMembershipsError()
	}

	previous, err := s.repo.ListOrganizationMemberships(ctx, userID, businessUnitID)
	if err != nil {
		return nil, err
	}

	if err = s.authorizeMembershipChange(ctx, &membershipAuthorization{
		actorID:        actorID,
		organizationID: organizationID,
		businessUnitID: businessUnitID,
		previous:       previous,
		desired:        organizationIDs,
	}); err != nil {
		return nil, err
	}

	memberships, err := s.repo.ReplaceOrganizationMemberships(
		ctx,
		tenantports.ReplaceOrganizationMembershipsRequest{
			ActorID:         actorID,
			UserID:          userID,
			BusinessUnitID:  businessUnitID,
			OrganizationIDs: organizationIDs,
		},
	)
	if err != nil {
		return nil, err
	}

	change := &membershipChange{
		actorID:        actorID,
		userID:         userID,
		organizationID: organizationID,
		businessUnitID: businessUnitID,
		before:         previous,
		after:          memberships,
	}
	s.recordMembershipChange(ctx, change)

	if err = s.revokeRemovedOrganizations(ctx, change); err != nil {
		s.l.Error(
			"failed to revoke access to organizations the user left",
			zap.String("userID", userID.String()),
			zap.Error(err),
		)
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
