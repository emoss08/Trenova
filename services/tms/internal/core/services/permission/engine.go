package permission

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	manifestVersion              = "1.0"
	cacheTTL                     = 30 * time.Minute
	slowPermissionCheckThreshold = 250 * time.Millisecond
)

type permissionLoadResult struct {
	perms               *repositories.CachedPermissions
	cacheHit            bool
	cacheLookupDuration time.Duration
	computeDuration     time.Duration
}

type permissionResultRequest struct {
	log       *zap.Logger
	start     time.Time
	load      permissionLoadResult
	req       *services.PermissionCheckRequest
	allowed   bool
	reason    string
	dataScope permission.DataScope
}

type Params struct {
	fx.In

	RoleRepository      repositories.RoleRepository
	RBACRepository      repositories.RBACRepository
	IAMRepository       repositories.IAMRepository
	APIKeyRepository    repositories.APIKeyRepository
	PermissionCacheRepo repositories.PermissionCacheRepository
	UserRepository      repositories.UserRepository
	Registry            *permission.Registry
	RouteRegistry       *permission.RouteRegistry
	Logger              *zap.Logger
}

type engine struct {
	roleRepo      repositories.RoleRepository
	rbacRepo      repositories.RBACRepository
	iamRepo       repositories.IAMRepository
	apiKeyRepo    repositories.APIKeyRepository
	cacheRepo     repositories.PermissionCacheRepository
	userRepo      repositories.UserRepository
	registry      *permission.Registry
	routeRegistry *permission.RouteRegistry
	l             *zap.Logger
}

//nolint:gocritic // this is dependency injection
func NewEngine(p Params) services.PermissionEngine {
	return &engine{
		roleRepo:      p.RoleRepository,
		rbacRepo:      p.RBACRepository,
		iamRepo:       p.IAMRepository,
		apiKeyRepo:    p.APIKeyRepository,
		cacheRepo:     p.PermissionCacheRepo,
		userRepo:      p.UserRepository,
		registry:      p.Registry,
		routeRegistry: p.RouteRegistry,
		l:             p.Logger.Named("service.permission-engine"),
	}
}

func (e *engine) Check(
	ctx context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	start := time.Now()
	log := e.l.With(
		zap.String("operation", "Check"),
		zap.String("principalID", req.PrincipalID.String()),
		zap.String("resource", req.Resource),
		zap.String("op", string(req.Operation)),
	)

	if req.PrincipalType == services.PrincipalTypeAPIKey {
		return e.checkAPIKeyPermission(ctx, req, start)
	}

	load, err := e.getOrComputePermissions(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		log.Error("failed to get permissions", zap.Error(err))
		e.logSlowPermissionCheck(log, start, load, false)
		return nil, err
	}

	return e.checkUserPermission(ctx, log, start, load, req)
}

func (e *engine) checkUserPermission(
	ctx context.Context,
	log *zap.Logger,
	start time.Time,
	load permissionLoadResult,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	perms := load.perms
	effectiveResource := e.registry.GetEffectiveResource(req.Resource)

	resourcePerms, ok := perms.Resources[effectiveResource]
	if !ok {
		return e.permissionResult(&permissionResultRequest{
			log:    log,
			start:  start,
			load:   load,
			req:    req,
			reason: "no_permission",
		}), nil
	}

	if slices.Contains(resourcePerms.Operations, string(req.Operation)) {
		result := e.permissionResult(&permissionResultRequest{
			log:       log,
			start:     start,
			load:      load,
			req:       req,
			allowed:   true,
			reason:    "allowed",
			dataScope: permission.DataScope(resourcePerms.DataScope),
		})
		if !result.Allowed {
			return result, nil
		}
		return e.applyAccessPolicies(ctx, req, result)
	}

	return e.permissionResult(&permissionResultRequest{
		log:    log,
		start:  start,
		load:   load,
		req:    req,
		reason: "no_permission",
	}), nil
}

func (e *engine) applyAccessPolicies(
	ctx context.Context,
	req *services.PermissionCheckRequest,
	result *services.PermissionCheckResult,
) (*services.PermissionCheckResult, error) {
	if e.iamRepo == nil {
		return result, nil
	}

	policies, err := e.iamRepo.ListEnabledAccessPolicies(ctx, repositories.IAMPolicyLookupRequest{
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		Resource:       e.registry.GetEffectiveResource(req.Resource),
		Operation:      req.Operation,
	})
	if err != nil {
		return nil, err
	}

	decision := evaluateAccessPolicies(policies, req)
	if decision == "" || decision == iam.PolicyEffectAllow {
		return result, nil
	}

	result.Allowed = false
	result.Reason = "iam_policy_denied"
	return result, nil
}

func evaluateAccessPolicies(
	policies []*iam.AccessPolicy,
	req *services.PermissionCheckRequest,
) iam.PolicyEffect {
	var currentPriority int
	var hasPriority bool
	var matchedAllow bool

	for _, policy := range policies {
		if !accessPolicyMatches(policy, req) {
			continue
		}
		if hasPriority && policy.Priority != currentPriority {
			if matchedAllow {
				return iam.PolicyEffectAllow
			}
			continue
		}
		if !hasPriority {
			hasPriority = true
			currentPriority = policy.Priority
		}
		if policy.Effect == iam.PolicyEffectDeny {
			return iam.PolicyEffectDeny
		}
		if policy.Effect == iam.PolicyEffectAllow {
			matchedAllow = true
		}
	}

	if matchedAllow {
		return iam.PolicyEffectAllow
	}
	return ""
}

func accessPolicyMatches(policy *iam.AccessPolicy, req *services.PermissionCheckRequest) bool {
	if policy == nil || len(policy.Conditions) == 0 {
		return true
	}

	for key, expected := range policy.Conditions {
		if !accessPolicyConditionMatches(key, expected, req) {
			return false
		}
	}
	return true
}

func accessPolicyConditionMatches(
	key string,
	expected string,
	req *services.PermissionCheckRequest,
) bool {
	normalizedKey := strings.TrimSpace(key)
	normalizedExpected := strings.TrimSpace(expected)

	switch normalizedKey {
	case "principalType":
		return string(req.PrincipalType) == normalizedExpected
	case "userId":
		return req.UserID.String() == normalizedExpected
	case "businessUnitId":
		return req.BusinessUnitID.String() == normalizedExpected
	case "organizationId":
		return req.OrganizationID.String() == normalizedExpected
	case "resourceId":
		return req.ResourceID != nil && req.ResourceID.String() == normalizedExpected
	case "ownerId":
		return req.ResourceAttributes.OwnerID.String() == normalizedExpected
	case "activeRoleId":
		return slices.Contains(req.ContextAttributes.ActiveRoleIDs, pulid.ID(normalizedExpected))
	case "riskDecision":
		return strings.EqualFold(req.ContextAttributes.RiskDecision, normalizedExpected)
	case "mfaRequired":
		return !strings.EqualFold(normalizedExpected, "true") ||
			req.ContextAttributes.MFAAuthenticatedAt > 0
	case "authenticatorAalMin":
		return intConditionAtLeast(req.ContextAttributes.AuthenticatorAAL, normalizedExpected)
	case "federationFalMin":
		return intConditionAtLeast(req.ContextAttributes.FederationFAL, normalizedExpected)
	default:
		return false
	}
}

func intConditionAtLeast(actual int, expected string) bool {
	var expectedValue int
	if _, err := fmt.Sscanf(expected, "%d", &expectedValue); err != nil {
		return false
	}
	return actual >= expectedValue
}

func (e *engine) permissionResult(p *permissionResultRequest) *services.PermissionCheckResult {
	if p.allowed {
		if result := enforceResourceAttributes(p.req, p.dataScope); result != nil {
			e.logSlowPermissionCheck(p.log, p.start, p.load, true)
			result.CacheHit = p.load.cacheHit
			result.CheckDuration = time.Since(p.start).Milliseconds()
			return result
		}
	}

	e.logSlowPermissionCheck(p.log, p.start, p.load, true)
	return &services.PermissionCheckResult{
		Allowed:       p.allowed,
		Reason:        p.reason,
		DataScope:     p.dataScope,
		CacheHit:      p.load.cacheHit,
		CheckDuration: time.Since(p.start).Milliseconds(),
	}
}

func (e *engine) logSlowPermissionCheck(
	log *zap.Logger,
	start time.Time,
	load permissionLoadResult,
	completed bool,
) {
	total := time.Since(start)
	if total <= slowPermissionCheckThreshold {
		return
	}

	log.Warn("slow permission check",
		zap.Duration("total_duration", total),
		zap.Duration("cache_lookup_duration", load.cacheLookupDuration),
		zap.Duration("permission_compute_duration", load.computeDuration),
		zap.Bool("cache_hit", load.cacheHit),
		zap.Bool("completed", completed),
	)
}

func (e *engine) CheckBatch(
	ctx context.Context,
	req *services.BatchPermissionCheckRequest,
) (*services.BatchPermissionCheckResult, error) {
	start := time.Now()

	results := make([]services.PermissionCheckResult, len(req.Checks))

	for i, check := range req.Checks {
		result, err := e.Check(ctx, &services.PermissionCheckRequest{
			PrincipalType:      req.PrincipalType,
			PrincipalID:        req.PrincipalID,
			UserID:             req.UserID,
			APIKeyID:           req.APIKeyID,
			BusinessUnitID:     req.BusinessUnitID,
			OrganizationID:     req.OrganizationID,
			Resource:           check.Resource,
			Operation:          check.Operation,
			ResourceID:         check.ResourceID,
			ResourceAttributes: check.ResourceAttributes,
			ContextAttributes:  req.ContextAttributes,
		})
		if err != nil {
			return nil, err
		}
		results[i] = *result
	}

	return &services.BatchPermissionCheckResult{
		Results:       results,
		CacheHit:      len(results) > 0 && results[0].CacheHit,
		CheckDuration: time.Since(start).Milliseconds(),
	}, nil
}

func enforceResourceAttributes(
	req *services.PermissionCheckRequest,
	dataScope permission.DataScope,
) *services.PermissionCheckResult {
	if result := enforceGlobalResourceAttributes(req, dataScope); result != nil {
		return result
	}

	attrs := req.ResourceAttributes

	switch dataScope {
	case permission.DataScopeOwn:
		if attrs.OwnerID.IsNotNil() && attrs.OwnerID != req.UserID {
			return deniedByABAC(dataScope, "abac_owner_scope")
		}
	case permission.DataScopeOrganization,
		permission.DataScopeBusinessUnit,
		permission.DataScopeAll:
	}

	return nil
}

func enforceGlobalResourceAttributes(
	req *services.PermissionCheckRequest,
	dataScope permission.DataScope,
) *services.PermissionCheckResult {
	attrs := req.ResourceAttributes

	if attrs.OrganizationID.IsNotNil() && attrs.OrganizationID != req.OrganizationID {
		return deniedByABAC(dataScope, "abac_organization_mismatch")
	}

	if attrs.BusinessUnitID.IsNotNil() && attrs.BusinessUnitID != req.BusinessUnitID {
		return deniedByABAC(dataScope, "abac_business_unit_mismatch")
	}

	if attrs.ActiveRoleID.IsNotNil() &&
		!slices.Contains(req.ContextAttributes.ActiveRoleIDs, attrs.ActiveRoleID) {
		return deniedByABAC(dataScope, "abac_active_role_required")
	}

	if strings.EqualFold(req.ContextAttributes.RiskDecision, "deny") {
		return deniedByABAC(dataScope, "abac_risk_denied")
	}

	return nil
}

func deniedByABAC(
	dataScope permission.DataScope,
	reason string,
) *services.PermissionCheckResult {
	return &services.PermissionCheckResult{
		Allowed:   false,
		Reason:    reason,
		DataScope: dataScope,
	}
}

func (e *engine) GetLightManifest(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*services.LightPermissionManifest, error) {
	log := e.l.With(
		zap.String("operation", "GetLightManifest"),
		zap.String("userID", userID.String()),
		zap.String("orgID", orgID.String()),
	)

	load, err := e.getOrComputePermissions(ctx, userID, orgID)
	if err != nil {
		log.Error("failed to get permissions", zap.Error(err))
		return nil, err
	}

	orgSummaries, err := e.getUserOrgSummaries(ctx, userID)
	if err != nil {
		log.Error("failed to get user organizations", zap.Error(err))
		return nil, err
	}
	expiresAt := timeutils.NowUnix() + int64(cacheTTL.Seconds())
	roleActivation, hasRoleActivation := authctx.GetSessionRoleActivation(ctx)
	activeRoleIDs := roleActivation.ActiveRoleIDs
	activeRoles := []services.RoleSummary{}
	requiresRoleActivation := false
	authorizedRoleIDs := []pulid.ID{}
	authorizedRoles := []services.RoleSummary{}
	if hasRoleActivation {
		authorizedRoles, err = e.getAuthorizedRoleSummaries(ctx, userID, orgID)
		if err != nil {
			log.Error("failed to get authorized roles", zap.Error(err))
			return nil, err
		}
		authorizedRoleIDs = roleSummaryIDs(authorizedRoles)
		activeRoles = activeRoleSummaries(activeRoleIDs, authorizedRoles)
		requiresRoleActivation = roleActivation.RequiresActivation &&
			len(activeRoleIDs) == 0 &&
			len(authorizedRoleIDs) > 0
	}

	perms := load.perms
	permissions := e.lightResourceBitmasks(perms)
	return e.newLightManifest(
		userID,
		orgID,
		permission.FieldSensitivity(perms.MaxSensitivity),
		permissions,
		orgSummaries,
		activeRoleIDs,
		authorizedRoleIDs,
		activeRoles,
		authorizedRoles,
		requiresRoleActivation,
		expiresAt,
	), nil
}

func (e *engine) getUserOrgSummaries(
	ctx context.Context,
	userID pulid.ID,
) ([]services.OrgSummary, error) {
	orgs, err := e.userRepo.GetUserOrganizationSummaries(ctx, userID)
	if err != nil {
		return nil, err
	}

	orgSummaries := make([]services.OrgSummary, len(orgs))
	for i, org := range orgs {
		orgSummaries[i] = services.OrgSummary{
			ID:   org.ID,
			Name: org.Name,
		}
	}

	return orgSummaries, nil
}

func (e *engine) getAuthorizedRoleSummaries(
	ctx context.Context,
	userID pulid.ID,
	orgID pulid.ID,
) ([]services.RoleSummary, error) {
	roles, err := e.rbacRepo.GetAuthorizedRoles(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	summaries := make([]services.RoleSummary, 0, len(roles))
	for _, role := range roles {
		if role != nil {
			summaries = append(summaries, services.NewRoleSummary(role))
		}
	}
	return summaries, nil
}

func roleSummaryIDs(roles []services.RoleSummary) []pulid.ID {
	roleIDs := make([]pulid.ID, len(roles))
	for i, role := range roles {
		roleIDs[i] = role.ID
	}
	return roleIDs
}

func activeRoleSummaries(
	activeRoleIDs []pulid.ID,
	authorizedRoles []services.RoleSummary,
) []services.RoleSummary {
	if len(activeRoleIDs) == 0 || len(authorizedRoles) == 0 {
		return []services.RoleSummary{}
	}

	byID := make(map[pulid.ID]services.RoleSummary, len(authorizedRoles))
	for _, role := range authorizedRoles {
		byID[role.ID] = role
	}

	activeRoles := make([]services.RoleSummary, 0, len(activeRoleIDs))
	for _, roleID := range activeRoleIDs {
		role, ok := byID[roleID]
		if ok {
			activeRoles = append(activeRoles, role)
		}
	}
	return activeRoles
}

func (*engine) lightResourceBitmasks(perms *repositories.CachedPermissions) map[string]uint32 {
	permissions := make(map[string]uint32)
	for resource, rp := range perms.Resources {
		var bitmask uint32
		for _, op := range rp.Operations {
			bitmask |= permission.OperationToBit[permission.Operation(op)]
		}
		permissions[resource] = bitmask
	}

	return permissions
}

func (e *engine) newLightManifest(
	userID, orgID pulid.ID,
	maxSensitivity permission.FieldSensitivity,
	permissions map[string]uint32,
	orgSummaries []services.OrgSummary,
	activeRoleIDs []pulid.ID,
	authorizedRoleIDs []pulid.ID,
	activeRoles []services.RoleSummary,
	authorizedRoles []services.RoleSummary,
	requiresRoleActivation bool,
	expiresAt int64,
) *services.LightPermissionManifest {
	manifest := &services.LightPermissionManifest{
		Version:                manifestVersion,
		UserID:                 userID,
		OrganizationID:         orgID,
		ActiveRoleIDs:          activeRoleIDs,
		AuthorizedRoleIDs:      authorizedRoleIDs,
		ActiveRoles:            activeRoles,
		AuthorizedRoles:        authorizedRoles,
		RequiresRoleActivation: requiresRoleActivation,
		MaxSensitivity:         maxSensitivity,
		Permissions:            permissions,
		RouteAccess:            e.computeRouteAccess(permissions),
		AvailableOrgs:          orgSummaries,
		ExpiresAt:              expiresAt,
	}
	manifest.Checksum = e.computeChecksum(manifest)
	return manifest
}

func (e *engine) GetResourcePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
	resource string,
) (*services.ResourcePermissionDetail, error) {
	log := e.l.With(
		zap.String("operation", "GetResourcePermissions"),
		zap.String("userID", userID.String()),
		zap.String("resource", resource),
	)

	def, ok := e.registry.Get(resource)
	if !ok {
		def, ok = e.registry.Get(e.registry.GetEffectiveResource(resource))
		if !ok {
			return nil, nil //nolint:nilnil // this is expected for non-registered resources
		}
	}

	load, err := e.getOrComputePermissions(ctx, userID, orgID)
	if err != nil {
		log.Error("failed to get permissions", zap.Error(err))
		return nil, err
	}

	perms := load.perms
	maxSensitivity := permission.FieldSensitivity(perms.MaxSensitivity)

	effectiveResource := e.registry.GetEffectiveResource(resource)
	resourcePerms, ok := perms.Resources[effectiveResource]
	if !ok {
		return &services.ResourcePermissionDetail{
			Resource:         resource,
			Operations:       []permission.Operation{},
			DataScope:        "",
			MaxSensitivity:   maxSensitivity,
			AccessibleFields: []string{},
		}, nil
	}

	ops := make([]permission.Operation, len(resourcePerms.Operations))
	for i, op := range resourcePerms.Operations {
		ops[i] = permission.Operation(op)
	}

	accessibleFields := e.getAccessibleFields(def, maxSensitivity)

	return &services.ResourcePermissionDetail{
		Resource:         resource,
		Operations:       ops,
		DataScope:        permission.DataScope(resourcePerms.DataScope),
		MaxSensitivity:   maxSensitivity,
		AccessibleFields: accessibleFields,
	}, nil
}

func (e *engine) InvalidateUser(ctx context.Context, userID, orgID pulid.ID) error {
	return e.cacheRepo.Delete(ctx, userID, orgID)
}

func (e *engine) GetEffectivePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*services.EffectivePermissions, error) {
	log := e.l.With(
		zap.String("operation", "GetEffectivePermissions"),
		zap.String("userID", userID.String()),
		zap.String("orgID", orgID.String()),
	)

	assignments, err := e.roleRepo.GetUserRoleAssignments(ctx, userID, orgID)
	if err != nil {
		log.Error("failed to get user role assignments", zap.Error(err))
		return nil, err
	}

	roleIDs := make([]pulid.ID, 0, len(assignments))
	for _, a := range assignments {
		if !a.IsExpired() {
			roleIDs = append(roleIDs, a.RoleID)
		}
	}

	roles, err := e.roleRepo.GetRolesWithInheritance(ctx, roleIDs)
	if err != nil {
		log.Error("failed to get roles with inheritance", zap.Error(err))
		return nil, err
	}

	return e.buildEffectivePermissions(userID, orgID, roles), nil
}

func (e *engine) SimulatePermissions(
	ctx context.Context,
	req *services.SimulatePermissionsRequest,
) (*services.EffectivePermissions, error) {
	log := e.l.With(
		zap.String("operation", "SimulatePermissions"),
		zap.String("userID", req.UserID.String()),
		zap.String("orgID", req.OrganizationID.String()),
	)

	assignments, err := e.roleRepo.GetUserRoleAssignments(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		log.Error("failed to get user role assignments", zap.Error(err))
		return nil, err
	}

	removeSet := make(map[pulid.ID]bool)
	for _, id := range req.RemoveRoleIDs {
		removeSet[id] = true
	}

	roleIDs := make([]pulid.ID, 0, len(assignments)+len(req.AddRoleIDs))
	for _, a := range assignments {
		if !a.IsExpired() && !removeSet[a.RoleID] {
			roleIDs = append(roleIDs, a.RoleID)
		}
	}
	roleIDs = append(roleIDs, req.AddRoleIDs...)

	roles, err := e.roleRepo.GetRolesWithInheritance(ctx, roleIDs)
	if err != nil {
		log.Error("failed to get roles with inheritance", zap.Error(err))
		return nil, err
	}

	return e.buildEffectivePermissions(req.UserID, req.OrganizationID, roles), nil
}

func (e *engine) checkAPIKeyPermission(
	ctx context.Context,
	req *services.PermissionCheckRequest,
	start time.Time,
) (*services.PermissionCheckResult, error) {
	perms, err := e.computeAPIKeyPermissions(
		ctx,
		req.APIKeyID,
		req.BusinessUnitID,
		req.OrganizationID,
	)
	if err != nil {
		return nil, err
	}

	effectiveResource := e.registry.GetEffectiveResource(req.Resource)
	resourcePerms, ok := perms.Resources[effectiveResource]
	if !ok {
		return &services.PermissionCheckResult{
			Allowed:       false,
			Reason:        "no_permission",
			DataScope:     "",
			CacheHit:      false,
			CheckDuration: time.Since(start).Milliseconds(),
		}, nil
	}

	if slices.Contains(resourcePerms.Operations, string(req.Operation)) {
		return &services.PermissionCheckResult{
			Allowed:       true,
			Reason:        "allowed",
			DataScope:     permission.DataScope(resourcePerms.DataScope),
			CacheHit:      false,
			CheckDuration: time.Since(start).Milliseconds(),
		}, nil
	}

	return &services.PermissionCheckResult{
		Allowed:       false,
		Reason:        "no_permission",
		DataScope:     "",
		CacheHit:      false,
		CheckDuration: time.Since(start).Milliseconds(),
	}, nil
}

func (e *engine) computeAPIKeyPermissions(
	ctx context.Context,
	apiKeyID, buID, orgID pulid.ID,
) (*repositories.CachedPermissions, error) {
	key, err := e.apiKeyRepo.GetByID(ctx, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	}, apiKeyID)
	if err != nil {
		return nil, err
	}

	resources := make(map[string]*repositories.CachedResourcePermission)
	for _, rp := range key.Permissions {
		e.mergeAPIKeyPermissionIntoCache(resources, rp)
	}

	return &repositories.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityRestricted),
		Resources:      resources,
		ExpiresAt:      timeutils.NowUnix() + int64(cacheTTL.Seconds()),
	}, nil
}

func (e *engine) getOrComputePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (permissionLoadResult, error) {
	result := permissionLoadResult{}
	_, hasRoleActivation := authctx.GetSessionRoleActivation(ctx)
	if hasRoleActivation {
		computeStart := time.Now()
		perms, err := e.computePermissions(ctx, userID, orgID)
		result.computeDuration = time.Since(computeStart)
		result.perms = perms
		return result, err
	}

	cacheStart := time.Now()
	cached, err := e.cacheRepo.Get(ctx, userID, orgID)
	result.cacheLookupDuration = time.Since(cacheStart)
	if err != nil {
		e.l.Warn("cache lookup failed, computing fresh", zap.Error(err))
	}

	if cached != nil && cached.ExpiresAt > timeutils.NowUnix() {
		result.perms = cached
		result.cacheHit = true
		return result, nil
	}

	computeStart := time.Now()
	perms, err := e.computePermissions(ctx, userID, orgID)
	result.computeDuration = time.Since(computeStart)
	if err != nil {
		return result, err
	}

	if cacheErr := e.cacheRepo.Set(ctx, userID, orgID, perms, cacheTTL); cacheErr != nil {
		e.l.Warn("failed to cache permissions", zap.Error(cacheErr))
	}

	result.perms = perms
	return result, nil
}

func (e *engine) computePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*repositories.CachedPermissions, error) {
	assignments, err := e.roleRepo.GetUserRoleAssignments(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	roleActivation, hasRoleActivation := authctx.GetSessionRoleActivation(ctx)
	if hasRoleActivation && roleActivation.RequiresActivation &&
		len(roleActivation.ActiveRoleIDs) == 0 {
		return &repositories.CachedPermissions{
			MaxSensitivity: string(permission.SensitivityPublic),
			Resources:      make(map[string]*repositories.CachedResourcePermission),
			ExpiresAt:      timeutils.NowUnix() + int64(cacheTTL.Seconds()),
		}, nil
	}

	roleIDs := activePermissionRoleIDs(assignments, roleActivation, hasRoleActivation)

	if len(roleIDs) == 0 {
		return &repositories.CachedPermissions{
			MaxSensitivity: string(permission.SensitivityPublic),
			Resources:      make(map[string]*repositories.CachedResourcePermission),
			ExpiresAt:      timeutils.NowUnix() + int64(cacheTTL.Seconds()),
		}, nil
	}

	roles, err := e.roleRepo.GetRolesWithInheritance(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	maxSensitivity := permission.SensitivityPublic
	resources := make(map[string]*repositories.CachedResourcePermission)

	for _, role := range roles {
		applyRoleSensitivity(role, &maxSensitivity)
		e.mergeRolePermissionsIntoCache(resources, role.Permissions)
	}

	return &repositories.CachedPermissions{
		MaxSensitivity: string(maxSensitivity),
		Resources:      resources,
		ExpiresAt:      timeutils.NowUnix() + int64(cacheTTL.Seconds()),
	}, nil
}

func activePermissionRoleIDs(
	assignments []*permission.UserRoleAssignment,
	roleActivation authctx.SessionRoleActivation,
	hasRoleActivation bool,
) []pulid.ID {
	authorized := make(map[pulid.ID]struct{}, len(assignments))
	roleIDs := make([]pulid.ID, 0, len(assignments))
	seenRoleIDs := make(map[pulid.ID]struct{}, len(assignments))
	for _, assignment := range assignments {
		if assignment.IsExpired() {
			continue
		}
		authorized[assignment.RoleID] = struct{}{}
		if hasRoleActivation {
			continue
		}
		if _, seen := seenRoleIDs[assignment.RoleID]; seen {
			continue
		}
		seenRoleIDs[assignment.RoleID] = struct{}{}
		roleIDs = append(roleIDs, assignment.RoleID)
	}
	if !hasRoleActivation {
		return roleIDs
	}

	activeRoleIDs := make([]pulid.ID, 0, len(roleActivation.ActiveRoleIDs))
	for _, id := range roleActivation.ActiveRoleIDs {
		if _, ok := authorized[id]; ok {
			activeRoleIDs = append(activeRoleIDs, id)
		}
	}
	return activeRoleIDs
}

func applyRoleSensitivity(
	role *permission.Role,
	maxSensitivity *permission.FieldSensitivity,
) {
	if role.MaxSensitivity.Level() > maxSensitivity.Level() {
		*maxSensitivity = role.MaxSensitivity
	}
}

func (e *engine) mergeRolePermissionsIntoCache(
	resources map[string]*repositories.CachedResourcePermission,
	rolePermissions []*permission.ResourcePermission,
) {
	for _, rp := range rolePermissions {
		e.mergeResourcePermissionIntoCache(resources, rp)
	}
}

func (e *engine) mergeAPIKeyPermissionIntoCache(
	resources map[string]*repositories.CachedResourcePermission,
	rp *apikey.Permission,
) {
	existing, ok := resources[rp.Resource]
	if !ok {
		resources[rp.Resource] = &repositories.CachedResourcePermission{
			Operations: e.opsToStrings(rp.Operations),
			DataScope:  string(rp.DataScope),
		}
		return
	}

	existing.Operations = e.mergeOpStrings(existing.Operations, rp.Operations)
	if rp.DataScope.IsMorePermissive(permission.DataScope(existing.DataScope)) {
		existing.DataScope = string(rp.DataScope)
	}
}

func (e *engine) mergeResourcePermissionIntoCache(
	resources map[string]*repositories.CachedResourcePermission,
	rp *permission.ResourcePermission,
) {
	existing, ok := resources[rp.Resource]
	if !ok {
		resources[rp.Resource] = &repositories.CachedResourcePermission{
			Operations: e.opsToStrings(rp.Operations),
			DataScope:  string(rp.DataScope),
		}
		return
	}

	existing.Operations = e.mergeOpStrings(existing.Operations, rp.Operations)
	if rp.DataScope.IsMorePermissive(permission.DataScope(existing.DataScope)) {
		existing.DataScope = string(rp.DataScope)
	}
}

func (e *engine) opsToStrings(ops []permission.Operation) []string {
	out := make([]string, len(ops))
	for i, op := range ops {
		out[i] = string(op)
	}
	return out
}

func (e *engine) mergeOpStrings(existing []string, incoming []permission.Operation) []string {
	opSet := make(map[string]struct{}, len(existing)+len(incoming))
	for _, op := range existing {
		opSet[op] = struct{}{}
	}
	for _, op := range incoming {
		opSet[string(op)] = struct{}{}
	}

	merged := make([]string, 0, len(opSet))
	for op := range opSet {
		merged = append(merged, op)
	}
	return merged
}

func (e *engine) buildEffectivePermissions(
	userID, orgID pulid.ID,
	roles []*permission.Role,
) *services.EffectivePermissions {
	maxSensitivity := permission.SensitivityPublic
	resources := make(map[string]services.EffectiveResourcePermission)

	roleSummaries := make([]services.RoleSummary, len(roles))
	for i, role := range roles {
		roleSummaries[i] = services.RoleSummary{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
			IsSystem:    role.IsSystem,
		}

		if role.MaxSensitivity.Level() > maxSensitivity.Level() {
			maxSensitivity = role.MaxSensitivity
		}

		for _, rp := range role.Permissions {
			existing, ok := resources[rp.Resource]
			if !ok {
				resources[rp.Resource] = services.EffectiveResourcePermission{
					Operations: rp.Operations,
					DataScope:  rp.DataScope,
					GrantedBy:  []string{role.Name},
				}
			} else {
				opSet := permission.NewOperationSet(existing.Operations...)
				for _, op := range rp.Operations {
					opSet.Add(op)
				}
				existing.Operations = opSet.ToSlice()

				if rp.DataScope.IsMorePermissive(existing.DataScope) {
					existing.DataScope = rp.DataScope
				}

				existing.GrantedBy = append(existing.GrantedBy, role.Name)
				resources[rp.Resource] = existing
			}
		}
	}

	return &services.EffectivePermissions{
		UserID:         userID,
		OrganizationID: orgID,
		Roles:          roleSummaries,
		MaxSensitivity: maxSensitivity,
		Resources:      resources,
	}
}

func (e *engine) computeChecksum(manifest *services.LightPermissionManifest) string {
	keys := make([]string, 0, len(manifest.Permissions))
	for k := range manifest.Permissions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	data := struct {
		ActiveRoleIDs          []pulid.ID
		AuthorizedRoleIDs      []pulid.ID
		RequiresRoleActivation bool
		MaxSensitivity         string
		Permissions            map[string]uint32
	}{
		ActiveRoleIDs:          manifest.ActiveRoleIDs,
		AuthorizedRoleIDs:      manifest.AuthorizedRoleIDs,
		RequiresRoleActivation: manifest.RequiresRoleActivation,
		MaxSensitivity:         string(manifest.MaxSensitivity),
		Permissions:            manifest.Permissions,
	}

	bytes, _ := sonic.Marshal(data)
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:8])
}

func (e *engine) computeRouteAccess(permissions map[string]uint32) map[string]bool {
	return e.routeRegistry.ComputeAccess(permissions)
}

func (e *engine) getAccessibleFields(
	def *permission.ResourceDefinition,
	maxSensitivity permission.FieldSensitivity,
) []string {
	var fields []string
	for field, sensitivity := range def.FieldSensitivities {
		if maxSensitivity.CanAccess(sensitivity) {
			fields = append(fields, field)
		}
	}
	sort.Strings(fields)
	return fields
}
