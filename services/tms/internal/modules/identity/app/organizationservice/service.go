//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package organizationservice

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           kernelports.DBConnection
	Repo         tenantports.OrganizationRepository
	SSORepo      tenantports.SSOConfigRepository
	AuditService auditports.AuditService
	Storage      kernelports.Client
	Config       *config.Config
	Validator    *Validator
	Encryption   *encryptionservice.Service
	Plans        platformplanports.PlanService `optional:"true"`
	ContentGate  contentscanports.Gate
}

type service struct {
	l            *zap.Logger
	db           kernelports.DBConnection
	repo         tenantports.OrganizationRepository
	ssoRepo      tenantports.SSOConfigRepository
	auditService auditports.AuditService
	storage      kernelports.Client
	storageCfg   *config.StorageConfig
	v            *Validator
	enc          *encryptionservice.Service
	plans        platformplanports.PlanService
	issuers      httpsafe.Policy
	contentGate  contentscanports.Gate
}

func New(p Params) tenantports.OrganizationService {
	return &service{
		l:            p.Logger.Named("service.organization"),
		db:           p.DB,
		repo:         p.Repo,
		ssoRepo:      p.SSORepo,
		storage:      p.Storage,
		auditService: p.AuditService,
		storageCfg:   p.Config.GetStorageConfig(),
		v:            p.Validator,
		enc:          p.Encryption,
		plans:        p.Plans,
		issuers:      p.Config.SSOIssuerPolicy(),
		contentGate:  p.ContentGate,
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

	if err := s.v.ValidateUpdate(ctx, entity); err != nil {
		return nil, err
	}

	var updatedEntity *tenant.Organization
	if err := s.db.WithTx(
		ctx,
		kernelports.TxOptions{},
		func(txCtx context.Context, _ bun.Tx) error {
			if err := s.guardCapabilityDisable(txCtx, entity); err != nil {
				return err
			}

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

func (s *service) UploadLogo(
	ctx context.Context,
	req *tenantports.UploadLogoRequest,
	userID pulid.ID,
) (*tenant.Organization, error) {
	log := s.l.With(
		zap.String("operation", "UploadLogo"),
		zap.String("userID", req.TenantInfo.UserID.String()),
	)

	if multiErr := s.validateLogoFile(req.File); multiErr != nil {
		return nil, multiErr
	}

	content, err := fileutils.ReadMultipart(req.File)
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to process uploaded logo").WithInternal(err)
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
	key := fileutils.GenerateStoragePath(
		req.OrganizationID.String(),
		"organization/logo",
		req.File.Filename,
	)

	if _, err = s.storage.Upload(ctx, &kernelports.UploadParams{
		Key:         key,
		ContentType: contentType,
		Size:        int64(len(content)),
		Body:        bytes.NewReader(content),
		Metadata: map[string]string{
			"original_name": req.File.Filename,
			"resource_type": "organization-logo",
			"resource_id":   req.OrganizationID.String(),
		},
	}); err != nil {
		return nil, errortypes.NewDatabaseError("Failed to upload organization logo").
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

	previousLogo := entity.LogoURL
	entity.LogoURL = key

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		_ = s.storage.Delete(ctx, key)
		return nil, err
	}

	if previousLogo != "" && !fileutils.IsExternalURL(previousLogo) && previousLogo != key {
		if delErr := s.storage.Delete(ctx, previousLogo); delErr != nil {
			s.l.Warn(
				"failed to delete previous organization logo",
				zap.String("orgID", req.OrganizationID.String()),
				zap.String("logoKey", previousLogo),
				zap.Error(delErr),
			)
		}
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
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
	req tenantports.GetLogoURLRequest,
) (*tenantports.GetLogoURLResponse, error) {
	entity, err := s.repo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
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

	if fileutils.IsExternalURL(entity.LogoURL) {
		return &tenantports.GetLogoURLResponse{URL: entity.LogoURL}, nil
	}

	url, err := s.storage.GetPresignedURL(ctx, &kernelports.PresignedURLParams{
		Key:    entity.LogoURL,
		Expiry: s.storageCfg.GetPresignedURLExpiry(),
	})
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to generate organization logo URL").
			WithInternal(err)
	}

	return &tenantports.GetLogoURLResponse{URL: url}, nil
}

func (s *service) DeleteLogo(
	ctx context.Context,
	req tenantports.DeleteLogoRequest,
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

	if org.LogoURL == "" {
		return org, nil
	}

	previousLogo := org.LogoURL
	updatedOrg, err := s.repo.ClearLogoURL(ctx, req.OrganizationID, org.Version)
	if err != nil {
		return nil, err
	}

	if !fileutils.IsExternalURL(previousLogo) {
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

	if !fileutils.IsSupportedImageContentType(file.Header.Get("Content-Type")) {
		multiErr.Add(
			"file",
			errortypes.ErrInvalidFormat,
			"Only image files are allowed for organization logos",
		)
	}

	if !fileutils.HasSupportedImageExtension(file.Filename) {
		multiErr.Add("file", errortypes.ErrInvalidFormat, "Unsupported logo file extension")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *service) GetMicrosoftSSOConfig(
	ctx context.Context,
	organizationID pulid.ID,
) (*tenantports.MicrosoftSSOConfig, error) {
	entity, err := s.ssoRepo.GetByOrganizationID(ctx, organizationID, tenant.SSOProviderAzureAD)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &tenantports.MicrosoftSSOConfig{
				OrganizationID: organizationID.String(),
			}, nil
		}

		return nil, err
	}

	return mapMicrosoftSSOConfig(entity), nil
}

//nolint:nestif // existing validation flow mirrors business rule nesting
func (s *service) UpsertMicrosoftSSOConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	cfg *tenantports.MicrosoftSSOConfig,
) (*tenantports.MicrosoftSSOConfig, error) {
	if cfg == nil {
		return nil, errortypes.NewValidationError(
			"config",
			errortypes.ErrRequired,
			"Microsoft Entra ID SSO configuration is required",
		)
	}
	if err := planservice.RequireCapability(
		ctx,
		s.plans,
		tenantInfo,
		platformplan.CapabilitySSO,
	); err != nil {
		return nil, err
	}

	tenantID := strings.TrimSpace(cfg.TenantID)
	clientID := strings.TrimSpace(cfg.ClientID)
	redirectURL := strings.TrimSpace(cfg.RedirectURL)

	existing, err := s.ssoRepo.GetByOrganizationID(ctx, tenantInfo.OrgID, tenant.SSOProviderAzureAD)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	rawClientSecret := strings.TrimSpace(cfg.ClientSecret)
	clientSecret := rawClientSecret
	if rawClientSecret == "" && !errortypes.IsNotFoundError(err) {
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

	if rawClientSecret != "" {
		clientSecret, err = s.enc.EncryptStringWithAAD(clientSecret, ssoSecretAAD(
			tenantInfo,
			string(tenant.SSOProviderAzureAD),
		))
		if err != nil {
			return nil, errortypes.NewBusinessError("Failed to encrypt Entra ID client secret").
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

func mapMicrosoftSSOConfig(entity *tenant.SSOConfig) *tenantports.MicrosoftSSOConfig {
	if entity == nil {
		return nil
	}

	allowedDomains := entity.AllowedDomains
	if allowedDomains == nil {
		allowedDomains = []string{}
	}

	return &tenantports.MicrosoftSSOConfig{
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

func ssoSecretAAD(tenantInfo pagination.TenantInfo, provider string) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeIAMOIDCClientSecret,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ResourceID:     provider,
	}
}

func (s *service) GetOktaSSOConfig(
	ctx context.Context,
	organizationID pulid.ID,
) (*tenantports.OktaSSOConfig, error) {
	entity, err := s.ssoRepo.GetByOrganizationID(ctx, organizationID, tenant.SSOProviderOkta)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &tenantports.OktaSSOConfig{
				OrganizationID: organizationID.String(),
				Scopes:         []string{"openid", "profile", "email"},
			}, nil
		}

		return nil, err
	}

	return mapOktaSSOConfig(entity), nil
}

//nolint:nestif // existing validation flow mirrors business rule nesting
func (s *service) UpsertOktaSSOConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	cfg *tenantports.OktaSSOConfig,
) (*tenantports.OktaSSOConfig, error) {
	if cfg == nil {
		return nil, errortypes.NewValidationError(
			"config",
			errortypes.ErrRequired,
			"Okta SSO configuration is required",
		)
	}
	if err := planservice.RequireCapability(
		ctx,
		s.plans,
		tenantInfo,
		platformplan.CapabilitySSO,
	); err != nil {
		return nil, err
	}

	issuerURL := strings.TrimSpace(cfg.IssuerURL)
	clientID := strings.TrimSpace(cfg.ClientID)
	redirectURL := strings.TrimSpace(cfg.RedirectURL)

	existing, err := s.ssoRepo.GetByOrganizationID(ctx, tenantInfo.OrgID, tenant.SSOProviderOkta)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	rawClientSecret := strings.TrimSpace(cfg.ClientSecret)
	clientSecret := rawClientSecret
	if rawClientSecret == "" && !errortypes.IsNotFoundError(err) {
		clientSecret = existing.OIDCClientSecret
	}

	if cfg.Enabled {
		multiErr := errortypes.NewMultiError()
		if issuerURL == "" {
			multiErr.Add("issuerUrl", errortypes.ErrRequired, "Issuer URL is required")
		}
		tenant.ValidateOIDCIssuerURL(multiErr, "issuerUrl", issuerURL, s.issuers)
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

	if rawClientSecret != "" {
		clientSecret, err = s.enc.EncryptStringWithAAD(clientSecret, ssoSecretAAD(
			tenantInfo,
			string(tenant.SSOProviderOkta),
		))
		if err != nil {
			return nil, errortypes.NewBusinessError("Failed to encrypt Okta client secret").
				WithInternal(err)
		}
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	entity := &tenant.SSOConfig{
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		Name:             "Okta",
		Provider:         tenant.SSOProviderOkta,
		Protocol:         tenant.SSOProtocolOIDC,
		Enabled:          cfg.Enabled,
		EnforceSSO:       cfg.EnforceSSO,
		AutoProvision:    false,
		AllowedDomains:   cfg.AllowedDomains,
		AttributeMap:     map[string]string{"email": "email"},
		OIDCIssuerURL:    issuerURL,
		OIDCClientID:     clientID,
		OIDCClientSecret: clientSecret,
		OIDCRedirectURL:  redirectURL,
		OIDCScopes:       scopes,
	}

	saved, err := s.ssoRepo.Save(ctx, entity)
	if err != nil {
		return nil, err
	}

	return mapOktaSSOConfig(saved), nil
}

func mapOktaSSOConfig(entity *tenant.SSOConfig) *tenantports.OktaSSOConfig {
	if entity == nil {
		return nil
	}

	allowedDomains := entity.AllowedDomains
	if allowedDomains == nil {
		allowedDomains = []string{}
	}

	scopes := entity.OIDCScopes
	if scopes == nil {
		scopes = []string{"openid", "profile", "email"}
	}

	return &tenantports.OktaSSOConfig{
		OrganizationID:   entity.OrganizationID.String(),
		Enabled:          entity.Enabled,
		EnforceSSO:       entity.EnforceSSO,
		IssuerURL:        entity.OIDCIssuerURL,
		ClientID:         entity.OIDCClientID,
		RedirectURL:      entity.OIDCRedirectURL,
		Scopes:           scopes,
		AllowedDomains:   allowedDomains,
		SecretConfigured: strings.TrimSpace(entity.OIDCClientSecret) != "",
	}
}

func microsoftTenantIDFromIssuer(issuerURL string) string {
	parts := strings.Split(strings.Trim(strings.TrimSpace(issuerURL), "/"), "/")
	if len(parts) < 2 {
		return ""
	}

	return parts[len(parts)-2]
}
