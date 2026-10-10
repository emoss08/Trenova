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
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/identity/ports/apikeyports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/hashutils"
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
	perms               *permissionports.CachedPermissions
	cacheHit            bool
	cacheLookupDuration time.Duration
	computeDuration     time.Duration
}

type permissionResultRequest struct {
	log       *zap.Logger
	start     time.Time
	load      permissionLoadResult
	req       *permissionports.PermissionCheckRequest
	allowed   bool
	reason    string
	dataScope permission.DataScope
}

type Params struct {
	fx.In

	RoleRepository        permissionports.RoleRepository
	RBACRepository        permissionports.RBACRepository
	IAMRepository         iamports.IAMRepository
	APIKeyRepository      apikeyports.APIKeyRepository
	PermissionCacheRepo   permissionports.PermissionCacheRepository
	AccessPolicyCacheRepo iamports.AccessPolicyCacheRepository
	UserRepository        tenantports.UserRepository
	AgentGrants           permissionports.RoleAgentGrantRepository  `optional:"true"`
	Delegated             permissionports.DelegatedPermissionSource `optional:"true"`
	Registry              *permission.Registry
	RouteRegistry         *permission.RouteRegistry
	Metrics               *metrics.Registry
	Logger                *zap.Logger
}

type engine struct {
	roleRepo      permissionports.RoleRepository
	rbacRepo      permissionports.RBACRepository
	iamRepo       iamports.IAMRepository
	apiKeyRepo    apikeyports.APIKeyRepository
	cacheRepo     permissionports.PermissionCacheRepository
	policyCache   iamports.AccessPolicyCacheRepository
	userRepo      tenantports.UserRepository
	agentGrants   permissionports.RoleAgentGrantRepository
	delegated     permissionports.DelegatedPermissionSource
	registry      *permission.Registry
	routeRegistry *permission.RouteRegistry
	metrics       *metrics.Registry
	l             *zap.Logger
}

//nolint:gocritic // this is dependency injection
func NewEngine(p Params) permissionports.PermissionEngine {
	return &engine{
		roleRepo:      p.RoleRepository,
		rbacRepo:      p.RBACRepository,
		iamRepo:       p.IAMRepository,
		apiKeyRepo:    p.APIKeyRepository,
		cacheRepo:     p.PermissionCacheRepo,
		policyCache:   p.AccessPolicyCacheRepo,
		userRepo:      p.UserRepository,
		agentGrants:   p.AgentGrants,
		delegated:     p.Delegated,
		registry:      p.Registry,
		routeRegistry: p.RouteRegistry,
		metrics:       p.Metrics,
		l:             p.Logger.Named("service.permission-engine"),
	}
}

func (e *engine) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	start := time.Now()
	log := e.checkLogger("Check", req)

	if req.PrincipalType == permissionports.PrincipalTypeAPIKey {
		return e.checkAPIKeyPermission(ctx, req, start)
	}

	if req.PrincipalType == permissionports.PrincipalTypeAgent {
		return e.checkAgentPermission(req, start), nil
	}

	load, err := e.getOrComputePermissions(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		log.Error("failed to get permissions", zap.Error(err))
		e.logSlowPermissionCheck(log, start, load, false)
		return nil, err
	}

	policies := e.newAccessPolicySource(req.OrganizationID, req.BusinessUnitID)
	return e.checkUserPermission(ctx, log, start, load, req, policies)
}

func (e *engine) checkLogger(
	operation string,
	req *permissionports.PermissionCheckRequest,
) *zap.Logger {
	return e.l.With(
		zap.String("operation", operation),
		zap.String("principalID", req.PrincipalID.String()),
		zap.String("resource", req.Resource),
		zap.String("op", string(req.Operation)),
	)
}

func (e *engine) checkAgentPermission(
	req *permissionports.PermissionCheckRequest,
	start time.Time,
) *permissionports.PermissionCheckResult {
	allowed := permission.IsAgentAllowed(permission.Resource(req.Resource), req.Operation)

	reason := "agent_denied"
	if allowed {
		reason = "agent_allowed"
	}

	return &permissionports.PermissionCheckResult{
		Allowed:       allowed,
		Reason:        reason,
		DataScope:     permission.DataScopeOrganization,
		CheckDuration: time.Since(start).Milliseconds(),
	}
}

func (e *engine) checkUserPermission(
	ctx context.Context,
	log *zap.Logger,
	start time.Time,
	load permissionLoadResult,
	req *permissionports.PermissionCheckRequest,
	policies accessPolicySource,
) (*permissionports.PermissionCheckResult, error) {
	resourcePerms, ok := e.resolveResourcePermission(load.perms, req.Resource, req.Operation)
	if !ok {
		return e.permissionResult(&permissionResultRequest{
			log:    log,
			start:  start,
			load:   load,
			req:    req,
			reason: "no_permission",
		}), nil
	}

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
	return e.applyAccessPolicies(ctx, req, result, policies)
}

func (e *engine) resolveResourcePermission(
	perms *permissionports.CachedPermissions,
	resource string,
	op permission.Operation,
) (*permissionports.CachedResourcePermission, bool) {
	if rp, ok := perms.Resources[resource]; ok &&
		slices.Contains(rp.Operations, string(op)) {
		return rp, true
	}

	effectiveResource := e.registry.GetEffectiveResource(resource)
	if effectiveResource != resource {
		if rp, ok := perms.Resources[effectiveResource]; ok &&
			slices.Contains(rp.Operations, string(op)) {
			return rp, true
		}
	}

	return nil, false
}

func (e *engine) applyAccessPolicies(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
	result *permissionports.PermissionCheckResult,
	policies accessPolicySource,
) (*permissionports.PermissionCheckResult, error) {
	if policies == nil {
		return result, nil
	}

	enabled, err := policies.enabledPolicies(
		ctx,
		e.registry.GetEffectiveResource(req.Resource),
		req.Operation,
	)
	if err != nil {
		return nil, err
	}

	decision := evaluateAccessPolicies(enabled, req)
	if decision == "" || decision == iam.PolicyEffectAllow {
		return result, nil
	}

	result.Allowed = false
	result.Reason = "iam_policy_denied"
	return result, nil
}

func (e *engine) newAccessPolicySource(orgID, buID pulid.ID) accessPolicySource {
	if e.iamRepo == nil {
		return nil
	}
	return &tenantAccessPolicySource{
		repo:  e.iamRepo,
		cache: e.policyCache,
		log:   e.l,
		req: iamports.IAMTenantPolicyLookupRequest{
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
	}
}

type accessPolicySource interface {
	enabledPolicies(
		ctx context.Context,
		resource string,
		op permission.Operation,
	) ([]*iam.AccessPolicy, error)
}

type accessPolicyKey struct {
	resource  string
	operation string
}

type tenantAccessPolicySource struct {
	repo   iamports.IAMRepository
	cache  iamports.AccessPolicyCacheRepository
	log    *zap.Logger
	req    iamports.IAMTenantPolicyLookupRequest
	loaded bool
	byKey  map[accessPolicyKey][]*iam.AccessPolicy
}

func (s *tenantAccessPolicySource) enabledPolicies(
	ctx context.Context,
	resource string,
	op permission.Operation,
) ([]*iam.AccessPolicy, error) {
	if !s.loaded {
		if err := s.load(ctx); err != nil {
			return nil, err
		}
	}
	return s.byKey[accessPolicyKey{resource: resource, operation: string(op)}], nil
}

func (s *tenantAccessPolicySource) load(ctx context.Context) error {
	policies, err := s.fetch(ctx)
	if err != nil {
		return err
	}

	s.byKey = make(map[accessPolicyKey][]*iam.AccessPolicy, len(policies))
	for _, policy := range policies {
		key := accessPolicyKey{resource: policy.Resource, operation: policy.Operation}
		s.byKey[key] = append(s.byKey[key], policy)
	}
	s.loaded = true
	return nil
}

func (s *tenantAccessPolicySource) fetch(ctx context.Context) ([]*iam.AccessPolicy, error) {
	if s.cache != nil {
		cached, found, err := s.cache.GetEnabled(ctx, s.req)
		if err != nil {
			s.log.Warn("access policy cache lookup failed, reading from database", zap.Error(err))
		} else if found {
			return cached, nil
		}
	}

	policies, err := s.repo.ListEnabledTenantAccessPolicies(ctx, s.req)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		if err = s.cache.SetEnabled(ctx, s.req, policies); err != nil {
			s.log.Warn("failed to cache access policies", zap.Error(err))
		}
	}
	return policies, nil
}

func evaluateAccessPolicies(
	policies []*iam.AccessPolicy,
	req *permissionports.PermissionCheckRequest,
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

func accessPolicyMatches(policy *iam.AccessPolicy, req *permissionports.PermissionCheckRequest) bool {
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
	req *permissionports.PermissionCheckRequest,
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

func (e *engine) permissionResult(p *permissionResultRequest) *permissionports.PermissionCheckResult {
	if p.allowed {
		if result := enforceResourceAttributes(p.req, p.dataScope); result != nil {
			e.logSlowPermissionCheck(p.log, p.start, p.load, true)
			result.CacheHit = p.load.cacheHit
			result.CheckDuration = time.Since(p.start).Milliseconds()
			return result
		}
	}

	e.logSlowPermissionCheck(p.log, p.start, p.load, true)
	return &permissionports.PermissionCheckResult{
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
	req *permissionports.BatchPermissionCheckRequest,
) (*permissionports.BatchPermissionCheckResult, error) {
	start := time.Now()

	results := make([]permissionports.PermissionCheckResult, len(req.Checks))
	if len(req.Checks) == 0 {
		return &permissionports.BatchPermissionCheckResult{
			Results:       results,
			CheckDuration: time.Since(start).Milliseconds(),
		}, nil
	}

	var err error
	switch req.PrincipalType {
	case permissionports.PrincipalTypeAPIKey:
		err = e.checkAPIKeyBatch(ctx, req, results)
	case permissionports.PrincipalTypeAgent:
		e.checkAgentBatch(req, results)
	default:
		err = e.checkUserBatch(ctx, req, results)
	}
	if err != nil {
		return nil, err
	}

	return &permissionports.BatchPermissionCheckResult{
		Results:       results,
		CacheHit:      results[0].CacheHit,
		CheckDuration: time.Since(start).Milliseconds(),
	}, nil
}

func batchCheckRequest(
	req *permissionports.BatchPermissionCheckRequest,
	check permissionports.ResourceOperationCheck,
) *permissionports.PermissionCheckRequest {
	return &permissionports.PermissionCheckRequest{
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
	}
}

func (e *engine) checkUserBatch(
	ctx context.Context,
	req *permissionports.BatchPermissionCheckRequest,
	results []permissionports.PermissionCheckResult,
) error {
	load, err := e.getOrComputePermissions(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		e.l.Error("failed to get permissions",
			zap.String("operation", "CheckBatch"),
			zap.String("principalID", req.PrincipalID.String()),
			zap.Error(err),
		)
		return err
	}

	policies := e.newAccessPolicySource(req.OrganizationID, req.BusinessUnitID)
	for i, check := range req.Checks {
		checkReq := batchCheckRequest(req, check)
		result, checkErr := e.checkUserPermission(
			ctx,
			e.checkLogger("CheckBatch", checkReq),
			time.Now(),
			load,
			checkReq,
			policies,
		)
		if checkErr != nil {
			return checkErr
		}
		results[i] = *result
	}
	return nil
}

func (e *engine) checkAPIKeyBatch(
	ctx context.Context,
	req *permissionports.BatchPermissionCheckRequest,
	results []permissionports.PermissionCheckResult,
) error {
	perms, err := e.computeAPIKeyPermissions(
		ctx,
		req.APIKeyID,
		req.BusinessUnitID,
		req.OrganizationID,
	)
	if err != nil {
		return err
	}

	for i, check := range req.Checks {
		results[i] = *e.resolveAPIKeyPermission(perms, batchCheckRequest(req, check), time.Now())
	}
	return nil
}

func (e *engine) checkAgentBatch(
	req *permissionports.BatchPermissionCheckRequest,
	results []permissionports.PermissionCheckResult,
) {
	for i, check := range req.Checks {
		results[i] = *e.checkAgentPermission(batchCheckRequest(req, check), time.Now())
	}
}

func enforceResourceAttributes(
	req *permissionports.PermissionCheckRequest,
	dataScope permission.DataScope,
) *permissionports.PermissionCheckResult {
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
	req *permissionports.PermissionCheckRequest,
	dataScope permission.DataScope,
) *permissionports.PermissionCheckResult {
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
) *permissionports.PermissionCheckResult {
	return &permissionports.PermissionCheckResult{
		Allowed:   false,
		Reason:    reason,
		DataScope: dataScope,
	}
}

func (e *engine) GetLightManifest(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*permissionports.LightPermissionManifest, error) {
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
	activeRoles := []permissionports.RoleSummary{}
	requiresRoleActivation := false
	authorizedRoleIDs := []pulid.ID{}
	authorizedRoles := []permissionports.RoleSummary{}
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
) ([]permissionports.OrgSummary, error) {
	orgs, err := e.userRepo.GetUserOrganizationSummaries(ctx, userID)
	if err != nil {
		return nil, err
	}

	orgSummaries := make([]permissionports.OrgSummary, len(orgs))
	for i, org := range orgs {
		orgSummaries[i] = permissionports.OrgSummary{
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
) ([]permissionports.RoleSummary, error) {
	roles, err := e.rbacRepo.GetAuthorizedRoles(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	summaries := make([]permissionports.RoleSummary, 0, len(roles))
	for _, role := range roles {
		if role != nil {
			summaries = append(summaries, permissionports.NewRoleSummary(role))
		}
	}

	counts, err := e.rbacRepo.CountRolePermissions(ctx, roleSummaryIDs(summaries))
	if err != nil {
		return nil, err
	}
	permissionports.ApplyRolePermissionCounts(summaries, counts)

	return summaries, nil
}

func roleSummaryIDs(roles []permissionports.RoleSummary) []pulid.ID {
	roleIDs := make([]pulid.ID, len(roles))
	for i, role := range roles {
		roleIDs[i] = role.ID
	}
	return roleIDs
}

func activeRoleSummaries(
	activeRoleIDs []pulid.ID,
	authorizedRoles []permissionports.RoleSummary,
) []permissionports.RoleSummary {
	if len(activeRoleIDs) == 0 || len(authorizedRoles) == 0 {
		return []permissionports.RoleSummary{}
	}

	byID := make(map[pulid.ID]permissionports.RoleSummary, len(authorizedRoles))
	for _, role := range authorizedRoles {
		byID[role.ID] = role
	}

	activeRoles := make([]permissionports.RoleSummary, 0, len(activeRoleIDs))
	for _, roleID := range activeRoleIDs {
		role, ok := byID[roleID]
		if ok {
			activeRoles = append(activeRoles, role)
		}
	}
	return activeRoles
}

func (*engine) lightResourceBitmasks(perms *permissionports.CachedPermissions) map[string]uint32 {
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
	orgSummaries []permissionports.OrgSummary,
	activeRoleIDs []pulid.ID,
	authorizedRoleIDs []pulid.ID,
	activeRoles []permissionports.RoleSummary,
	authorizedRoles []permissionports.RoleSummary,
	requiresRoleActivation bool,
	expiresAt int64,
) *permissionports.LightPermissionManifest {
	manifest := &permissionports.LightPermissionManifest{
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
) (*permissionports.ResourcePermissionDetail, error) {
	log := e.l.With(
		zap.String("operation", "GetResourcePermissions"),
		zap.String("userID", userID.String()),
		zap.String("resource", resource),
	)

	if _, ok := e.registry.Get(resource); !ok {
		if _, ok = e.registry.Get(e.registry.GetEffectiveResource(resource)); !ok {
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
		return &permissionports.ResourcePermissionDetail{
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

	// No field allowlist: the registry knows only the fields a resource
	// classifies itself, and every reader treats a non-empty list as the
	// complete set. Built from the classified fields alone, it withheld every
	// unclassified one, such as a move's miles, though MaxSensitivity, which
	// each reader checks first, already allows them.
	return &permissionports.ResourcePermissionDetail{
		Resource:       resource,
		Operations:     ops,
		DataScope:      permission.DataScope(resourcePerms.DataScope),
		MaxSensitivity: maxSensitivity,
	}, nil
}

func (e *engine) InvalidateUser(ctx context.Context, userID, orgID pulid.ID) error {
	return e.cacheRepo.Delete(ctx, userID, orgID)
}

func (e *engine) GetEffectivePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*permissionports.EffectivePermissions, error) {
	log := e.l.With(
		zap.String("operation", "GetEffectivePermissions"),
		zap.String("userID", userID.String()),
		zap.String("orgID", orgID.String()),
	)

	start := time.Now()
	defer func() {
		e.metrics.Permission.RecordCompute(
			metrics.PermissionPathEffective, time.Since(start),
		)
	}()

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
	e.metrics.Permission.RecordClosureSize(len(roles))

	return e.buildEffectivePermissions(userID, orgID, roles), nil
}

func (e *engine) SimulatePermissions(
	ctx context.Context,
	req *permissionports.SimulatePermissionsRequest,
) (*permissionports.EffectivePermissions, error) {
	log := e.l.With(
		zap.String("operation", "SimulatePermissions"),
		zap.String("userID", req.UserID.String()),
		zap.String("orgID", req.OrganizationID.String()),
	)

	start := time.Now()
	defer func() {
		e.metrics.Permission.RecordCompute(
			metrics.PermissionPathSimulation, time.Since(start),
		)
	}()

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
	e.metrics.Permission.RecordClosureSize(len(roles))

	return e.buildEffectivePermissions(req.UserID, req.OrganizationID, roles), nil
}

func (e *engine) checkAPIKeyPermission(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
	start time.Time,
) (*permissionports.PermissionCheckResult, error) {
	perms, err := e.computeAPIKeyPermissions(
		ctx,
		req.APIKeyID,
		req.BusinessUnitID,
		req.OrganizationID,
	)
	if err != nil {
		return nil, err
	}

	return e.resolveAPIKeyPermission(perms, req, start), nil
}

func (e *engine) resolveAPIKeyPermission(
	perms *permissionports.CachedPermissions,
	req *permissionports.PermissionCheckRequest,
	start time.Time,
) *permissionports.PermissionCheckResult {
	resourcePerms, ok := e.resolveResourcePermission(perms, req.Resource, req.Operation)
	if !ok {
		return &permissionports.PermissionCheckResult{
			Allowed:       false,
			Reason:        "no_permission",
			DataScope:     "",
			CacheHit:      false,
			CheckDuration: time.Since(start).Milliseconds(),
		}
	}

	return &permissionports.PermissionCheckResult{
		Allowed:       true,
		Reason:        "allowed",
		DataScope:     permission.DataScope(resourcePerms.DataScope),
		CacheHit:      false,
		CheckDuration: time.Since(start).Milliseconds(),
	}
}

func (e *engine) computeAPIKeyPermissions(
	ctx context.Context,
	apiKeyID, buID, orgID pulid.ID,
) (*permissionports.CachedPermissions, error) {
	key, err := e.apiKeyRepo.GetByID(ctx, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	}, apiKeyID)
	if err != nil {
		return nil, err
	}

	resources := make(map[string]*permissionports.CachedResourcePermission)
	for _, rp := range key.Permissions {
		e.mergeAPIKeyPermissionIntoCache(resources, rp)
	}

	return &permissionports.CachedPermissions{
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
	if e.delegated != nil {
		perms, delegated, err := e.delegated.DelegatedPermissions(ctx, userID, orgID)
		if err != nil {
			return result, err
		}
		if delegated {
			result.perms = perms
			return result, nil
		}
	}

	key := permissionports.PermissionCacheKey{
		UserID:  userID,
		OrgID:   orgID,
		Variant: permissionCacheVariant(ctx),
	}

	cacheStart := time.Now()
	cached, err := e.cacheRepo.Get(ctx, key)
	result.cacheLookupDuration = time.Since(cacheStart)

	outcome := metrics.PermissionCacheMiss
	switch {
	case err != nil:
		outcome = metrics.PermissionCacheError
		e.l.Warn("cache lookup failed, computing fresh", zap.Error(err))
	case cached != nil && cached.ExpiresAt > timeutils.NowUnix():
		outcome = metrics.PermissionCacheHit
	}
	e.metrics.Permission.RecordCacheLookup(outcome, result.cacheLookupDuration)

	if outcome == metrics.PermissionCacheHit {
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
	e.metrics.Permission.RecordCompute(metrics.PermissionPathCached, result.computeDuration)

	if cacheErr := e.cacheRepo.Set(ctx, key, perms, cacheTTL); cacheErr != nil {
		e.l.Warn("failed to cache permissions", zap.Error(cacheErr))
	}

	result.perms = perms
	return result, nil
}

func permissionCacheVariant(ctx context.Context) string {
	activation, ok := authctx.GetSessionRoleActivation(ctx)
	if !ok {
		return permissionports.PermissionCacheVariantAllRoles
	}

	ids := make([]string, 0, len(activation.ActiveRoleIDs))
	for _, id := range activation.ActiveRoleIDs {
		ids = append(ids, id.String())
	}
	sort.Strings(ids)
	ids = slices.Compact(ids)

	return "roles:" + hashutils.SHA256Hex(strings.Join(ids, ","))
}

func (e *engine) computePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*permissionports.CachedPermissions, error) {
	assignments, err := e.roleRepo.GetUserRoleAssignments(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	roleActivation, hasRoleActivation := authctx.GetSessionRoleActivation(ctx)
	if hasRoleActivation && roleActivation.RequiresActivation &&
		len(roleActivation.ActiveRoleIDs) == 0 {
		return &permissionports.CachedPermissions{
			MaxSensitivity: string(permission.SensitivityPublic),
			Resources:      make(map[string]*permissionports.CachedResourcePermission),
			ExpiresAt:      timeutils.NowUnix() + int64(cacheTTL.Seconds()),
		}, nil
	}

	roleIDs := activePermissionRoleIDs(assignments, roleActivation, hasRoleActivation)

	if len(roleIDs) == 0 {
		return &permissionports.CachedPermissions{
			MaxSensitivity: string(permission.SensitivityPublic),
			Resources:      make(map[string]*permissionports.CachedResourcePermission),
			ExpiresAt:      timeutils.NowUnix() + int64(cacheTTL.Seconds()),
		}, nil
	}

	roles, err := e.roleRepo.GetRolesWithInheritance(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	e.metrics.Permission.RecordClosureSize(len(roles))

	maxSensitivity := permission.SensitivityPublic
	resources := make(map[string]*permissionports.CachedResourcePermission)

	for _, role := range roles {
		applyRoleSensitivity(role, &maxSensitivity)
		e.mergeRolePermissionsIntoCache(resources, role.Permissions)
	}

	agentIDs, err := e.grantedAgentIDs(ctx, orgID, roles)
	if err != nil {
		return nil, err
	}

	return &permissionports.CachedPermissions{
		MaxSensitivity: string(maxSensitivity),
		Resources:      resources,
		AgentIDs:       agentIDs,
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
	resources map[string]*permissionports.CachedResourcePermission,
	rolePermissions []*permission.ResourcePermission,
) {
	for _, rp := range rolePermissions {
		e.mergeResourcePermissionIntoCache(resources, rp)
	}
}

func (e *engine) mergeAPIKeyPermissionIntoCache(
	resources map[string]*permissionports.CachedResourcePermission,
	rp *apikey.Permission,
) {
	existing, ok := resources[rp.Resource]
	if !ok {
		resources[rp.Resource] = &permissionports.CachedResourcePermission{
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
	resources map[string]*permissionports.CachedResourcePermission,
	rp *permission.ResourcePermission,
) {
	existing, ok := resources[rp.Resource]
	if !ok {
		resources[rp.Resource] = &permissionports.CachedResourcePermission{
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
) *permissionports.EffectivePermissions {
	maxSensitivity := permission.SensitivityPublic
	resources := make(map[string]permissionports.EffectiveResourcePermission)

	roleSummaries := make([]permissionports.RoleSummary, len(roles))
	for i, role := range roles {
		roleSummaries[i] = permissionports.RoleSummary{
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
				resources[rp.Resource] = permissionports.EffectiveResourcePermission{
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

	return &permissionports.EffectivePermissions{
		UserID:         userID,
		OrganizationID: orgID,
		Roles:          roleSummaries,
		MaxSensitivity: maxSensitivity,
		Resources:      resources,
	}
}

type resourceBitmask struct {
	Resource string
	Bitmask  uint32
}

func (e *engine) computeChecksum(manifest *permissionports.LightPermissionManifest) string {
	keys := make([]string, 0, len(manifest.Permissions))
	for k := range manifest.Permissions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	permissions := make([]resourceBitmask, len(keys))
	for i, k := range keys {
		permissions[i] = resourceBitmask{Resource: k, Bitmask: manifest.Permissions[k]}
	}

	data := struct {
		ActiveRoleIDs          []pulid.ID
		AuthorizedRoleIDs      []pulid.ID
		RequiresRoleActivation bool
		MaxSensitivity         string
		Permissions            []resourceBitmask
	}{
		ActiveRoleIDs:          manifest.ActiveRoleIDs,
		AuthorizedRoleIDs:      manifest.AuthorizedRoleIDs,
		RequiresRoleActivation: manifest.RequiresRoleActivation,
		MaxSensitivity:         string(manifest.MaxSensitivity),
		Permissions:            permissions,
	}

	bytes, _ := sonic.Marshal(data)
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:8])
}

func (e *engine) computeRouteAccess(permissions map[string]uint32) map[string]bool {
	return e.routeRegistry.ComputeAccess(permissions)
}
