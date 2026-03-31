package organizationservice

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.OrganizationRepository
	SSORepo      repositories.SSOConfigRepository
	AuditService services.AuditService
	Storage      storage.Client
	Config       *config.Config
	Validator    *Validator
	Encryption   *encryptionservice.Service
}

type service struct {
	l            *zap.Logger
	repo         repositories.OrganizationRepository
	ssoRepo      repositories.SSOConfigRepository
	auditService services.AuditService
	storage      storage.Client
	storageCfg   *config.StorageConfig
	v            *Validator
	enc          *encryptionservice.Service
}

func New(p Params) services.OrganizationService {
	return &service{
		l:            p.Logger.Named("service.organization"),
		repo:         p.Repo,
		ssoRepo:      p.SSORepo,
		storage:      p.Storage,
		auditService: p.AuditService,
		storageCfg:   p.Config.GetStorageConfig(),
		v:            p.Validator,
		enc:          p.Encryption,
	}
}

func (s *service) GetByID(
	ctx context.Context,
	req repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *service) Update(
	ctx context.Context,
	entity *tenant.Organization,
) (*tenant.Organization, error) {
	log := s.l.With(zap.String("operation", "Update"), zap.String("orgID", entity.ID.String()))

	if err := s.v.ValidateUpdate(ctx, entity); err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update organization", zap.Error(err))
		return nil, mapOrganizationUniqueConstraint(err)
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

func (s *service) UploadLogo(
	ctx context.Context,
	req *services.UploadLogoRequest,
	userID pulid.ID,
) (*tenant.Organization, error) {
	log := s.l.With(
		zap.String("operation", "UploadLogo"),
		zap.String("userID", req.TenantInfo.UserID.String()),
	)

	if multiErr := s.validateLogoFile(req.File); multiErr != nil {
		return nil, multiErr
	}

	file, err := req.File.Open()
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to process uploaded logo").WithInternal(err)
	}
	defer file.Close()

	contentType := req.File.Header.Get("Content-Type")
	ext := strings.ToLower(filepath.Ext(req.File.Filename))
	key := fmt.Sprintf(
		"%s/organization/logo/%s%s",
		req.OrganizationID.String(),
		uuid.NewString(),
		ext,
	)

	if _, err = s.storage.Upload(ctx, &storage.UploadParams{
		Key:         key,
		ContentType: contentType,
		Size:        req.File.Size,
		Body:        file,
		Metadata: map[string]string{
			"original_name": req.File.Filename,
			"resource_type": "organization-logo",
			"resource_id":   req.OrganizationID.String(),
		},
	}); err != nil {
		return nil, errortypes.NewDatabaseError("Failed to upload organization logo").
			WithInternal(err)
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		_ = s.storage.Delete(ctx, key)
		return nil, err
	}

	previousLogo := entity.LogoURL
	entity.LogoURL = key

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		_ = s.storage.Delete(ctx, key)
		return nil, err
	}

	if previousLogo != "" && !isExternalLogoURL(previousLogo) && previousLogo != key {
		if delErr := s.storage.Delete(ctx, previousLogo); delErr != nil {
			s.l.Warn(
				"failed to delete previous organization logo",
				zap.String("orgID", req.OrganizationID.String()),
				zap.String("logoKey", previousLogo),
				zap.Error(delErr),
			)
		}
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceOrganization,
		ResourceID:     updatedEntity.GetResourceID(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(entity),
		OrganizationID: entity.ID,
		BusinessUnitID: entity.BusinessUnitID,
	}, auditservice.WithComment("Organization updated")); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *service) GetLogoURL(
	ctx context.Context,
	req services.GetLogoURLRequest,
) (*services.GetLogoURLResponse, error) {
	entity, err := s.repo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, err
	}

	if entity.LogoURL == "" {
		return nil, errortypes.NewNotFoundError("Organization logo not found")
	}

	if isExternalLogoURL(entity.LogoURL) {
		return &services.GetLogoURLResponse{URL: entity.LogoURL}, nil
	}

	url, err := s.storage.GetPresignedURL(ctx, &storage.PresignedURLParams{
		Key:    entity.LogoURL,
		Expiry: s.storageCfg.GetPresignedURLExpiry(),
	})
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to generate organization logo URL").
			WithInternal(err)
	}

	return &services.GetLogoURLResponse{URL: url}, nil
}

func (s *service) DeleteLogo(
	ctx context.Context,
	req services.DeleteLogoRequest,
) (*tenant.Organization, error) {
	org, err := s.repo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, err
	}

	if org.LogoURL == "" {
		return org, nil
	}

	previousLogo := org.LogoURL
	updatedOrg, err := s.repo.ClearLogoURL(ctx, req.OrganizationID, org.Version)
	if err != nil {
		return nil, err
	}

	if !isExternalLogoURL(previousLogo) {
		if delErr := s.storage.Delete(ctx, previousLogo); delErr != nil {
			s.l.Warn(
				"failed to delete organization logo object after clearing logo URL",
				zap.String("orgID", req.OrganizationID.String()),
				zap.String("logoKey", previousLogo),
				zap.Error(delErr),
			)
		}
	}

	return updatedOrg, nil
}

func (s *service) validateLogoFile(file *multipart.FileHeader) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if file == nil {
		multiErr.Add("file", errortypes.ErrRequired, "Logo file is required")
		return multiErr
	}

	if file.Size == 0 {
		multiErr.Add("file", errortypes.ErrRequired, "Logo file cannot be empty")
	}

	if file.Size > s.storageCfg.GetMaxFileSize() {
		multiErr.Add("file", errortypes.ErrInvalidLength, "Logo file exceeds maximum allowed size")
	}

	contentType := strings.ToLower(file.Header.Get("Content-Type"))
	allowedMIMETypes := []string{"image/jpeg", "image/png", "image/webp"}
	if contentType != "" &&
		contentType != "application/octet-stream" &&
		!slices.Contains(allowedMIMETypes, contentType) {
		multiErr.Add(
			"file",
			errortypes.ErrInvalidFormat,
			"Only image files are allowed for organization logos",
		)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExtensions := []string{".jpg", ".jpeg", ".png", ".webp"}
	if ext == "" || !slices.Contains(allowedExtensions, ext) {
		multiErr.Add("file", errortypes.ErrInvalidFormat, "Unsupported logo file extension")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func isExternalLogoURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func (s *service) GetMicrosoftSSOConfig(
	ctx context.Context,
	organizationID pulid.ID,
) (*services.MicrosoftSSOConfig, error) {
	entity, err := s.ssoRepo.GetByOrganizationID(ctx, organizationID, tenant.SSOProviderAzureAD)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &services.MicrosoftSSOConfig{
				OrganizationID: organizationID.String(),
			}, nil
		}

		return nil, err
	}

	return mapMicrosoftSSOConfig(entity), nil
}

func (s *service) UpsertMicrosoftSSOConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	cfg *services.MicrosoftSSOConfig,
) (*services.MicrosoftSSOConfig, error) {
	if cfg == nil {
		return nil, errortypes.NewValidationError(
			"config",
			errortypes.ErrRequired,
			"Microsoft SSO configuration is required",
		)
	}

	tenantID := strings.TrimSpace(cfg.TenantID)
	clientID := strings.TrimSpace(cfg.ClientID)
	redirectURL := strings.TrimSpace(cfg.RedirectURL)

	existing, err := s.ssoRepo.GetByOrganizationID(ctx, tenantInfo.OrgID, tenant.SSOProviderAzureAD)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	clientSecret := strings.TrimSpace(cfg.ClientSecret)
	if clientSecret == "" && !errortypes.IsNotFoundError(err) {
		clientSecret = existing.OIDCClientSecret
	}

	if cfg.Enabled {
		multiErr := errortypes.NewMultiError()
		if tenantID == "" {
			multiErr.Add("tenantId", errortypes.ErrRequired, "Tenant ID is required")
		}
		if clientID == "" {
			multiErr.Add("clientId", errortypes.ErrRequired, "Client ID is required")
		}
		if clientSecret == "" {
			multiErr.Add("clientSecret", errortypes.ErrRequired, "Client secret is required")
		}
		if redirectURL == "" {
			multiErr.Add("redirectUrl", errortypes.ErrRequired, "Redirect URL is required")
		}
		if multiErr.HasErrors() {
			return nil, multiErr
		}
	}

	if clientSecret != "" {
		clientSecret, err = s.enc.EncryptString(clientSecret)
		if err != nil {
			return nil, errortypes.NewBusinessError("Failed to encrypt Microsoft client secret").
				WithInternal(err)
		}
	}

	entity := &tenant.SSOConfig{
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		Name:             "Microsoft Entra ID",
		Provider:         tenant.SSOProviderAzureAD,
		Protocol:         tenant.SSOProtocolOIDC,
		Enabled:          cfg.Enabled,
		EnforceSSO:       cfg.EnforceSSO,
		AutoProvision:    false,
		AllowedDomains:   cfg.AllowedDomains,
		AttributeMap:     map[string]string{"email": "email"},
		OIDCIssuerURL:    microsoftIssuerURL(tenantID),
		OIDCClientID:     clientID,
		OIDCClientSecret: clientSecret,
		OIDCRedirectURL:  redirectURL,
		OIDCScopes:       []string{"openid", "profile", "email"},
	}

	saved, err := s.ssoRepo.Save(ctx, entity)
	if err != nil {
		return nil, err
	}

	return mapMicrosoftSSOConfig(saved), nil
}

func mapMicrosoftSSOConfig(entity *tenant.SSOConfig) *services.MicrosoftSSOConfig {
	if entity == nil {
		return nil
	}

	allowedDomains := entity.AllowedDomains
	if allowedDomains == nil {
		allowedDomains = []string{}
	}

	return &services.MicrosoftSSOConfig{
		OrganizationID:   entity.OrganizationID.String(),
		Enabled:          entity.Enabled,
		EnforceSSO:       entity.EnforceSSO,
		TenantID:         microsoftTenantIDFromIssuer(entity.OIDCIssuerURL),
		ClientID:         entity.OIDCClientID,
		RedirectURL:      entity.OIDCRedirectURL,
		AllowedDomains:   allowedDomains,
		SecretConfigured: strings.TrimSpace(entity.OIDCClientSecret) != "",
	}
}

func microsoftIssuerURL(tenantID string) string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)
}

func microsoftTenantIDFromIssuer(issuerURL string) string {
	parts := strings.Split(strings.Trim(strings.TrimSpace(issuerURL), "/"), "/")
	if len(parts) < 2 {
		return ""
	}

	return parts[len(parts)-2]
}
