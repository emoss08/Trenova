package apikeyservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/apikeyports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger   *zap.Logger
	Repo     apikeyports.APIKeyRepository
	Registry *permission.Registry
	Auditor  permissionports.SecurityAuditor
	Engine   permissionports.PermissionEngine

	Config *config.Config
	Plans  platformplanports.PlanService `optional:"true"`
}

type Service struct {
	l        *zap.Logger
	repo     apikeyports.APIKeyRepository
	registry *permission.Registry
	auditor  permissionports.SecurityAuditor
	engine   permissionports.PermissionEngine
	cfg      *config.Config
	plans    platformplanports.PlanService
}

func New(p Params) *Service {
	return &Service{
		l:        p.Logger.Named("service.api-key"),
		repo:     p.Repo,
		registry: p.Registry,
		auditor:  p.Auditor,
		engine:   p.Engine,
		cfg:      p.Config,
		plans:    p.Plans,
	}
}

func (s *Service) ListAPIKeys(
	ctx context.Context,
	req *apikeyports.ListAPIKeysRequest,
) (*pagination.ListResult[apikeyports.APIKeyResponse], error) {
	keys, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list api keys").WithInternal(err)
	}

	items := make([]apikeyports.APIKeyResponse, 0, len(keys.Items))
	for _, key := range keys.Items {
		items = append(items, s.mapAPIKeyResponse(key))
	}

	return &pagination.ListResult[apikeyports.APIKeyResponse]{
		Items: items,
		Total: keys.Total,
	}, nil
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *apikeyports.ListAPIKeyConnectionRequest,
) (*pagination.CursorListResult[*apikey.Key], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) ComputePermissionScope(perms []*apikey.Permission) string {
	return s.computePermissionScope(perms)
}

func (s *Service) GetAPIKeyEntity(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*apikey.Key, error) {
	return s.repo.GetByID(ctx, tenantInfo, id)
}

func (s *Service) CreateAPIKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *apikeyports.CreateAPIKeyRequest,
	userID pulid.ID,
) (*apikeyports.APIKeySecretResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := planservice.RequireCapability(
		ctx,
		s.plans,
		tenantInfo,
		platformplan.CapabilityAPIKeys,
	); err != nil {
		return nil, err
	}

	expiresAt, err := s.resolveCreateExpiry(req.ExpiresAt)
	if err != nil {
		return nil, err
	}

	if err = s.enforceCreatorKeyLimit(ctx, tenantInfo, userID); err != nil {
		return nil, err
	}

	normalizedPermissions, err := s.normalizePermissions(req.Permissions)
	if err != nil {
		return nil, err
	}

	if err = s.validateWithinActor(ctx, userID, tenantInfo.OrgID, normalizedPermissions); err != nil {
		return nil, err
	}

	generated, err := apikey.GenerateAPIKeySecret()
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to generate api key").WithInternal(err)
	}

	key := &apikey.Key{
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		Name:           req.Name,
		Description:    req.Description,
		KeyPrefix:      generated.Prefix,
		SecretHash:     generated.Hash,
		SecretSalt:     generated.Salt,
		Status:         apikey.StatusActive,
		ExpiresAt:      expiresAt,
		CreatedByID:    userID,
	}

	perms := s.toDomainPermissions(key, normalizedPermissions)
	if err = s.repo.CreateWithPermissions(ctx, key, perms); err != nil {
		return nil, errortypes.NewBusinessError("failed to create api key").WithInternal(err)
	}
	key.Permissions = perms

	resp := s.mapAPIKeyResponse(key)
	s.recordChange(ctx, &keyChange{
		tenantInfo: tenantInfo,
		actorID:    userID,
		key:        key,
		operation:  permission.OpCreate,
		after:      resp,
		comment:    "API key created",
	})

	return &apikeyports.APIKeySecretResponse{
		APIKeyResponse: resp,
		Token:          generated.Token(),
	}, nil
}

func (s *Service) GetAPIKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*apikeyports.APIKeyResponse, error) {
	key, err := s.repo.GetByID(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}

	resp := s.mapAPIKeyResponse(key)
	return &resp, nil
}

func (s *Service) UpdateAPIKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	req *apikeyports.UpdateAPIKeyRequest,
) (*apikeyports.APIKeyResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	normalizedPermissions, err := s.normalizePermissions(req.Permissions)
	if err != nil {
		return nil, err
	}

	if err = s.validateWithinActor(
		ctx,
		tenantInfo.UserID,
		tenantInfo.OrgID,
		normalizedPermissions,
	); err != nil {
		return nil, err
	}

	key, err := s.repo.GetByID(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}
	before := s.mapAPIKeyResponse(key)

	expiresAt, err := s.resolveUpdateExpiry(req.ExpiresAt)
	if err != nil {
		return nil, err
	}

	key.Name = req.Name
	key.Description = req.Description
	key.ExpiresAt = expiresAt
	perms := s.toDomainPermissions(key, normalizedPermissions)
	if err = s.repo.UpdateWithPermissions(ctx, key, perms); err != nil {
		return nil, errortypes.NewBusinessError("failed to update api key").WithInternal(err)
	}
	key.Permissions = perms

	resp := s.mapAPIKeyResponse(key)
	s.recordChange(ctx, &keyChange{
		tenantInfo: tenantInfo,
		actorID:    tenantInfo.UserID,
		key:        key,
		operation:  permission.OpUpdate,
		before:     before,
		after:      resp,
		comment:    "API key updated",
	})
	return &resp, nil
}

func (s *Service) RotateAPIKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*apikeyports.APIKeySecretResponse, error) {
	if err := planservice.RequireCapability(
		ctx,
		s.plans,
		tenantInfo,
		platformplan.CapabilityAPIKeys,
	); err != nil {
		return nil, err
	}

	key, err := s.repo.GetByID(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}

	before := s.mapAPIKeyResponse(key)

	generated, err := apikey.GenerateAPIKeySecret()
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to rotate api key").WithInternal(err)
	}

	key.KeyPrefix = generated.Prefix
	key.SecretHash = generated.Hash
	key.SecretSalt = generated.Salt
	key.Status = apikey.StatusActive
	key.RevokedAt = 0
	key.RevokedByID = ""
	key.LastUsedAt = 0
	key.LastUsedIP = ""
	key.LastUsedUserAgent = ""
	if err = s.repo.Update(ctx, key); err != nil {
		return nil, errortypes.NewBusinessError("failed to rotate api key").WithInternal(err)
	}

	resp := s.mapAPIKeyResponse(key)
	s.recordChange(ctx, &keyChange{
		tenantInfo: tenantInfo,
		actorID:    tenantInfo.UserID,
		key:        key,
		operation:  permission.OpUpdate,
		before:     before,
		after:      resp,
		comment:    "API key secret rotated",
	})
	return &apikeyports.APIKeySecretResponse{
		APIKeyResponse: resp,
		Token:          generated.Token(),
	}, nil
}

func (s *Service) RevokeAPIKey(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id, userID pulid.ID,
) (*apikeyports.APIKeyResponse, error) {
	key, err := s.repo.GetByID(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}

	before := s.mapAPIKeyResponse(key)

	key.Status = apikey.StatusRevoked
	key.RevokedByID = userID
	key.RevokedAt = timeutils.NowUnix()
	if err = s.repo.Update(ctx, key); err != nil {
		return nil, errortypes.NewBusinessError("failed to revoke api key").WithInternal(err)
	}

	resp := s.mapAPIKeyResponse(key)
	s.recordChange(ctx, &keyChange{
		tenantInfo: tenantInfo,
		actorID:    userID,
		key:        key,
		operation:  permission.OpUpdate,
		before:     before,
		after:      resp,
		comment:    "API key revoked",
	})
	return &resp, nil
}

type AllowedResourceCategory struct {
	Category  string                           `json:"category"`
	Resources []*permission.ResourceDefinition `json:"resources"`
}

func (s *Service) GetAllowedResources() []AllowedResourceCategory {
	categoryMap := make(map[string][]*permission.ResourceDefinition)

	for resource, allowedOps := range runtimePolicy {
		def, ok := s.registry.Get(resource)
		if !ok {
			continue
		}

		filtered := &permission.ResourceDefinition{
			Resource:           def.Resource,
			DisplayName:        def.DisplayName,
			Description:        def.Description,
			Category:           def.Category,
			ParentResource:     def.ParentResource,
			DefaultSensitivity: def.DefaultSensitivity,
		}

		for _, opDef := range def.Operations {
			if slices.Contains(allowedOps, opDef.Operation) {
				filtered.Operations = append(filtered.Operations, opDef)
			}
		}

		categoryMap[def.Category] = append(categoryMap[def.Category], filtered)
	}

	categories := make([]string, 0, len(categoryMap))
	for cat := range categoryMap {
		categories = append(categories, cat)
	}
	slices.Sort(categories)

	result := make([]AllowedResourceCategory, 0, len(categories))
	for _, cat := range categories {
		resources := categoryMap[cat]
		slices.SortFunc(resources, func(a, b *permission.ResourceDefinition) int {
			return strings.Compare(a.DisplayName, b.DisplayName)
		})
		result = append(result, AllowedResourceCategory{
			Category:  cat,
			Resources: resources,
		})
	}

	return result
}

func (s *Service) toDomainPermissions(
	key *apikey.Key,
	inputs []apikeyports.APIKeyPermissionInput,
) []*apikey.Permission {
	perms := make([]*apikey.Permission, 0, len(inputs))
	for _, input := range inputs {
		perms = append(perms, &apikey.Permission{
			APIKeyID:       key.ID,
			BusinessUnitID: key.BusinessUnitID,
			OrganizationID: key.OrganizationID,
			Resource:       input.Resource,
			Operations:     input.Operations,
			DataScope:      input.DataScope,
		})
	}
	return perms
}

func (s *Service) mapAPIKeyResponse(key *apikey.Key) apikeyports.APIKeyResponse {
	resp := apikeyports.APIKeyResponse{
		ID:             key.ID.String(),
		BusinessUnitID: key.BusinessUnitID.String(),
		OrganizationID: key.OrganizationID.String(),
		Name:           key.Name,
		Description:    key.Description,
		KeyPrefix:      key.KeyPrefix,
		Status:         string(key.Status),
		ExpiresAt:      key.ExpiresAt,
		LastUsedAt:     key.LastUsedAt,
		CreatedAt:      key.CreatedAt,
		UpdatedAt:      key.UpdatedAt,
	}

	resp.Permissions = make([]apikeyports.APIKeyPermissionInput, 0, len(key.Permissions))
	for _, permission := range key.Permissions {
		resp.Permissions = append(resp.Permissions, apikeyports.APIKeyPermissionInput{
			Resource:   permission.Resource,
			Operations: permission.Operations,
			DataScope:  permission.DataScope,
		})
	}

	resp.PermissionScope = s.computePermissionScope(key.Permissions)

	return resp
}

func (s *Service) computePermissionScope(perms []*apikey.Permission) string {
	permMap := make(map[string][]permission.Operation, len(perms))
	for _, p := range perms {
		permMap[p.Resource] = p.Operations
	}

	for resource, requiredOps := range runtimePolicy {
		if _, registered := s.registry.Get(resource); !registered {
			continue
		}

		grantedOps, ok := permMap[resource]
		if !ok {
			return "restricted"
		}
		for _, op := range requiredOps {
			if !slices.Contains(grantedOps, op) {
				return "restricted"
			}
		}
	}

	return "full"
}

func normalizeDataScope(scope permission.DataScope) permission.DataScope {
	if scope == "" {
		return permission.DataScopeOrganization
	}
	return scope
}

func (s *Service) normalizePermissions(
	inputs []apikeyports.APIKeyPermissionInput,
) ([]apikeyports.APIKeyPermissionInput, error) {
	normalized := make([]apikeyports.APIKeyPermissionInput, 0, len(inputs))
	for _, input := range inputs {
		normalized = append(normalized, apikeyports.APIKeyPermissionInput{
			Resource:   strings.TrimSpace(input.Resource),
			Operations: permission.Normalize(input.Operations),
			DataScope:  normalizeDataScope(input.DataScope),
		})
	}

	if err := s.validateRuntimePermissions(normalized); err != nil {
		return nil, err
	}

	return normalized, nil
}

func (s *Service) validateRuntimePermissions(inputs []apikeyports.APIKeyPermissionInput) error {
	me := errortypes.NewMultiError()

	for _, input := range inputs {
		resource := input.Resource
		if resource == "" {
			me.Add("permissions", errortypes.ErrRequired, "Resource is required")
			continue
		}

		allowedOperations, ok := runtimePolicy[resource]
		if !ok {
			me.Add(
				"permissions",
				errortypes.ErrInvalid,
				"Resource \"{0}\" is not available for API keys", resource,
			)
			continue
		}

		def, ok := s.registry.Get(resource)
		if !ok {
			me.Add(
				"permissions",
				errortypes.ErrInvalid,
				"Resource \"{0}\" is not registered", resource,
			)
			continue
		}

		if !input.DataScope.IsValid() {
			me.Add(
				"permissions",
				errortypes.ErrInvalid,
				"Resource \"{0}\" has an invalid data scope", resource,
			)
		}

		for _, operation := range input.Operations {
			if !slices.Contains(allowedOperations, operation) {
				me.Add(
					"permissions",
					errortypes.ErrInvalid,
					"Operation \"{0}\" is not available for API keys", operation,
				)
				continue
			}

			if !resourceSupportsOperation(def, operation) {
				me.Add(
					"permissions",
					errortypes.ErrInvalid,
					"Resource \"{0}\" does not support operation \"{1}\"", resource, operation,
				)
			}
		}
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) validateWithinActor(
	ctx context.Context,
	actorID, orgID pulid.ID,
	inputs []apikeyports.APIKeyPermissionInput,
) error {
	held, err := s.engine.GetEffectivePermissions(ctx, actorID, orgID)
	if err != nil {
		return err
	}

	for _, input := range inputs {
		violation, violated := held.DelegationViolation(
			input.Resource,
			input.Operations,
			input.DataScope,
		)
		if violated {
			return violation.FieldError("permissions")
		}
	}

	return nil
}

func resourceSupportsOperation(
	def *permission.ResourceDefinition,
	operation permission.Operation,
) bool {
	for _, opDef := range def.Operations {
		if opDef.Operation == operation {
			return true
		}
	}

	return false
}

func (s *Service) enforceCreatorKeyLimit(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) error {
	limit := s.cfg.Security.APIToken.MaxTokensPerUser
	if limit <= 0 {
		return nil
	}

	count, err := s.repo.CountActiveByCreator(ctx, tenantInfo, userID)
	if err != nil {
		return errortypes.NewBusinessError("failed to enforce api key limit").WithInternal(err)
	}

	if count >= limit {
		return errortypes.NewBusinessError("maximum active api keys reached")
	}

	return nil
}

func (s *Service) resolveCreateExpiry(expiresAt int64) (int64, error) {
	now := timeutils.NowUnix()
	if expiresAt == 0 && s.cfg.Security.APIToken.DefaultExpiry > 0 {
		expiresAt = now + int64(s.cfg.Security.APIToken.DefaultExpiry.Seconds())
	}

	return s.validateExpiry(expiresAt)
}

func (s *Service) resolveUpdateExpiry(expiresAt int64) (int64, error) {
	return s.validateExpiry(expiresAt)
}

func (s *Service) validateExpiry(expiresAt int64) (int64, error) {
	if expiresAt == 0 {
		return 0, nil
	}

	now := timeutils.NowUnix()
	if expiresAt <= now {
		return 0, errortypes.NewValidationError(
			"expiresAt",
			errortypes.ErrInvalid,
			"Expiration must be in the future",
		)
	}

	maxExpiry := s.cfg.Security.APIToken.MaxExpiry
	if maxExpiry > 0 && expiresAt > now+int64(maxExpiry.Seconds()) {
		return 0, errortypes.NewValidationError(
			"expiresAt",
			errortypes.ErrInvalid,
			"Expiration exceeds the maximum allowed lifetime",
		)
	}

	return expiresAt, nil
}
