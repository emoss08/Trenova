package permission

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/auditmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/operationbits"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/routeaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/apikeyports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
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

	RoleRepository      permissionports.RoleRepository
	RBACRepository      permissionports.RBACRepository
	IAMRepository       iamports.IAMRepository
	AccessSettings      iamports.AccessSettingsRepository
	APIKeyRepository    apikeyports.APIKeyRepository
	PermissionCacheRepo permissionports.PermissionCacheRepository
	UserRepository      tenantports.UserRepository
	AgentGrants         permissionports.RoleAgentGrantRepository
	AgentRoles          permissionports.AgentRoleSource
	Registry            *permissioncatalog.Registry
	RouteRegistry       *routeaccess.RouteRegistry
	DatabaseConfig      *databaseconfig.Config
	Metrics             *metrics.Registry
	Logger              *zap.Logger
}

type engine struct {
	roleRepo          permissionports.RoleRepository
	rbacRepo          permissionports.RBACRepository
	iamRepo           iamports.IAMRepository
	settings          iamports.AccessSettingsRepository
	apiKeyRepo        apikeyports.APIKeyRepository
	cacheRepo         permissionports.PermissionCacheRepository
	userRepo          tenantports.UserRepository
	agentGrants       permissionports.RoleAgentGrantRepository
	agentRoles        permissionports.AgentRoleSource
	registry          *permissioncatalog.Registry
	routeRegistry     *routeaccess.RouteRegistry
	databaseEnforcing bool
	metrics           *metrics.Registry
	l                 *zap.Logger
}

//nolint:gocritic // this is dependency injection
func NewEngine(p Params) permissionports.PermissionEngine {
	return &engine{
		roleRepo:          p.RoleRepository,
		rbacRepo:          p.RBACRepository,
		iamRepo:           p.IAMRepository,
		settings:          p.AccessSettings,
		apiKeyRepo:        p.APIKeyRepository,
		cacheRepo:         p.PermissionCacheRepo,
		userRepo:          p.UserRepository,
		agentGrants:       p.AgentGrants,
		agentRoles:        p.AgentRoles,
		registry:          p.Registry,
		routeRegistry:     p.RouteRegistry,
		databaseEnforcing: p.DatabaseConfig.RLS.Enforced(),
		metrics:           p.Metrics,
		l:                 p.Logger.Named("service.permission-engine"),
	}
}

func (e *engine) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	start := time.Now()
	log := e.checkLogger("Check", req)
	req = requestInSession(ctx, req)
	ctx = e.withoutActivatedRoles(
		ctx,
		req.PrincipalType,
		req.UserID,
		req.OrganizationID,
		req.BusinessUnitID,
	)

	if req.PrincipalType == permissionports.PrincipalTypeAPIKey {
		return e.checkAPIKeyPermission(ctx, req, start)
	}

	if req.PrincipalType == permissionports.PrincipalTypeAgent {
		return e.checkAgentPermission(
			ctx,
			req,
			start,
			e.newAccessPolicySource(req.OrganizationID, req.BusinessUnitID),
		)
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

func (e *engine) withoutActivatedRoles(
	ctx context.Context,
	principalType kernelports.PrincipalType,
	userID, orgID, buID pulid.ID,
) context.Context {
	if principalType != permissionports.PrincipalTypeUser && principalType != "" {
		return ctx
	}
	session, ok := authctx.GetSessionAttributes(ctx)
	if !ok || session.UserID != userID || session.AuthenticatorAAL != authctx.UnprovenAssurance {
		return ctx
	}
	if _, bound := authctx.GetSessionRoleActivation(ctx); bound {
		return ctx
	}

	settings, err := e.settings.Get(dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}), pagination.TenantInfo{OrgID: orgID, BuID: buID})
	if err != nil {
		e.l.Warn(
			"could not read whether the organization requires role activation; requiring it",
			zap.String("organizationID", orgID.String()),
			zap.Error(err),
		)
		return authctx.WithSessionRoleActivation(ctx, nil, true)
	}
	if !settings.RequireRoleActivation {
		return ctx
	}

	return authctx.WithSessionRoleActivation(ctx, nil, true)
}

func requestInSession(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) *permissionports.PermissionCheckRequest {
	attributes, ok := sessionContextAttributes(ctx, req.PrincipalType, req.UserID)
	if !ok || req.ContextAttributes.AuthenticatorAAL > 0 {
		return req
	}

	scoped := *req
	scoped.ContextAttributes = attributes
	return &scoped
}

func sessionContextAttributes(
	ctx context.Context,
	principalType kernelports.PrincipalType,
	userID pulid.ID,
) (permissionports.RequestContextAttributes, bool) {
	if principalType != permissionports.PrincipalTypeUser && principalType != "" &&
		principalType != authctx.PrincipalTypeCaptureDevice {
		return permissionports.RequestContextAttributes{}, false
	}

	session, ok := authctx.GetSessionAttributes(ctx)
	if !ok || session.UserID != userID {
		return permissionports.RequestContextAttributes{}, false
	}

	return permissionports.RequestContextAttributes{
		ActiveRoleIDs:         session.ActiveRoleIDs,
		AuthenticatorAAL:      session.AuthenticatorAAL,
		FederationFAL:         session.FederationFAL,
		MFAAuthenticatedAt:    session.MFAAuthenticatedAt,
		LastReauthenticatedAt: session.LastReauthenticatedAt,
		RiskDecision:          session.RiskDecision,
	}, true
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
	if rp, ok := e.usableGrant(perms, resource, op); ok {
		return rp, true
	}

	effectiveResource := e.registry.GetEffectiveResource(resource)
	if effectiveResource != resource {
		return e.usableGrant(perms, effectiveResource, op)
	}

	return nil, false
}

func (e *engine) usableGrant(
	perms *permissionports.CachedPermissions,
	resource string,
	op permission.Operation,
) (*permissionports.CachedResourcePermission, bool) {
	rp, ok := perms.Resources[resource]
	if !ok || !slices.Contains(rp.Operations, string(op)) {
		return nil, false
	}
	if !e.databaseEnforcing &&
		e.registry.EnforcedByDatabase(resource, permission.DataScope(rp.DataScope)) {
		return nil, false
	}

	return rp, true
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

	enabled, err := e.policiesFor(ctx, req, policies)
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

func (e *engine) policiesFor(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
	policies accessPolicySource,
) ([]*iam.AccessPolicy, error) {
	effective := e.registry.GetEffectiveResource(req.Resource)
	enabled, err := policies.enabledPolicies(ctx, effective, req.Operation)
	if err != nil || effective == req.Resource {
		return enabled, err
	}

	own, err := policies.enabledPolicies(ctx, req.Resource, req.Operation)
	if err != nil {
		return nil, err
	}

	merged := slices.Concat(own, enabled)
	slices.SortStableFunc(merged, func(a, b *iam.AccessPolicy) int {
		return cmp.Compare(b.Priority, a.Priority)
	})

	return merged, nil
}

func (e *engine) newAccessPolicySource(orgID, buID pulid.ID) accessPolicySource {
	return &tenantAccessPolicySource{
		repo: e.iamRepo,
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
	policies, err := s.repo.ListEnabledTenantAccessPolicies(ctx, s.req)
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

const (
	conditionMFARequired         = "mfaRequired"
	conditionAuthenticatorAALMin = "authenticatorAalMin"
	conditionFederationFALMin    = "federationFalMin"
)

func evaluateAccessPolicies(
	policies []*iam.AccessPolicy,
	req *permissionports.PermissionCheckRequest,
) iam.PolicyEffect {
	var currentPriority int
	var hasPriority bool
	var matchedAllow bool

	for _, policy := range policies {
		effect, applies := accessPolicyEffect(policy, req)
		if !applies {
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
		if effect == iam.PolicyEffectDeny {
			return iam.PolicyEffectDeny
		}
		if effect == iam.PolicyEffectAllow {
			matchedAllow = true
		}
	}

	if matchedAllow {
		return iam.PolicyEffectAllow
	}
	return ""
}

func accessPolicyEffect(
	policy *iam.AccessPolicy,
	req *permissionports.PermissionCheckRequest,
) (iam.PolicyEffect, bool) {
	if policy == nil || !accessPolicyMatches(policy, req) {
		return "", false
	}

	required, met := accessPolicyRequirements(policy, req)
	switch {
	case !met:
		return iam.PolicyEffectDeny, true
	case required && policy.Effect == iam.PolicyEffectDeny:
		return "", false
	default:
		return policy.Effect, true
	}
}

func accessPolicyMatches(
	policy *iam.AccessPolicy,
	req *permissionports.PermissionCheckRequest,
) bool {
	for key, expected := range policy.Conditions {
		if isAccessPolicyRequirement(key, expected) {
			continue
		}
		if !accessPolicyConditionMatches(key, expected, req) {
			return false
		}
	}
	return true
}

func accessPolicyRequirements(
	policy *iam.AccessPolicy,
	req *permissionports.PermissionCheckRequest,
) (required, met bool) {
	met = true
	for key, expected := range policy.Conditions {
		if !isAccessPolicyRequirement(key, expected) {
			continue
		}
		required = true
		if req.ContextAttributes.AuthenticatorAAL > 0 &&
			!accessPolicyRequirementMet(key, expected, req) {
			met = false
		}
	}
	return required, met
}

func isAccessPolicyRequirement(key, expected string) bool {
	switch strings.TrimSpace(key) {
	case conditionMFARequired:
		return strings.EqualFold(strings.TrimSpace(expected), "true")
	case conditionAuthenticatorAALMin, conditionFederationFALMin:
		return true
	default:
		return false
	}
}

func accessPolicyRequirementMet(
	key string,
	expected string,
	req *permissionports.PermissionCheckRequest,
) bool {
	attributes := req.ContextAttributes
	normalizedExpected := strings.TrimSpace(expected)

	switch strings.TrimSpace(key) {
	case conditionMFARequired:
		return attributes.MFAAuthenticatedAt > 0
	case conditionAuthenticatorAALMin:
		return intConditionAtLeast(attributes.AuthenticatorAAL, normalizedExpected)
	case conditionFederationFALMin:
		return intConditionAtLeast(attributes.FederationFAL, normalizedExpected)
	default:
		return false
	}
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
	case "principalId":
		return req.PrincipalID.String() == normalizedExpected
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
	case conditionMFARequired:
		return true
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

func (e *engine) permissionResult(
	p *permissionResultRequest,
) *permissionports.PermissionCheckResult {
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

	if attributes, ok := sessionContextAttributes(ctx, req.PrincipalType, req.UserID); ok &&
		req.ContextAttributes.AuthenticatorAAL == 0 {
		scoped := *req
		scoped.ContextAttributes = attributes
		req = &scoped
	}
	ctx = e.withoutActivatedRoles(
		ctx,
		req.PrincipalType,
		req.UserID,
		req.OrganizationID,
		req.BusinessUnitID,
	)

	var err error
	switch req.PrincipalType {
	case permissionports.PrincipalTypeAPIKey:
		err = e.checkAPIKeyBatch(ctx, req, results)
	case permissionports.PrincipalTypeAgent:
		err = e.checkAgentBatch(ctx, req, results)
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
	perms, creatorID, err := e.computeAPIKeyPermissions(
		ctx,
		req.APIKeyID,
		req.BusinessUnitID,
		req.OrganizationID,
	)
	if err != nil {
		return err
	}

	policies := e.newAccessPolicySource(req.OrganizationID, req.BusinessUnitID)
	for i, check := range req.Checks {
		checkReq := batchCheckRequest(req, check)
		result := e.resolveAPIKeyPermission(perms, checkReq, creatorID, time.Now())
		if result.Allowed {
			if result, err = e.applyAccessPolicies(ctx, checkReq, result, policies); err != nil {
				return err
			}
		}
		results[i] = *result
	}
	return nil
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
			bitmask |= operationbits.ByOperation[permission.Operation(op)]
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
	if !e.registered(resource) {
		return nil, nil //nolint:nilnil // this is expected for non-registered resources
	}

	load, err := e.getOrComputePermissions(ctx, userID, orgID)
	if err != nil {
		e.l.Error("failed to get permissions",
			zap.String("operation", "GetResourcePermissions"),
			zap.String("userID", userID.String()),
			zap.String("resource", resource),
			zap.Error(err),
		)
		return nil, err
	}

	return e.resourceDetail(load.perms, resource), nil
}

func (e *engine) GetResourcePermissionsBatch(
	ctx context.Context,
	userID, orgID pulid.ID,
	resources []string,
) (map[string]*permissionports.ResourcePermissionDetail, error) {
	details := make(map[string]*permissionports.ResourcePermissionDetail, len(resources))
	if len(resources) == 0 {
		return details, nil
	}

	load, err := e.getOrComputePermissions(ctx, userID, orgID)
	if err != nil {
		e.l.Error("failed to get permissions",
			zap.String("operation", "GetResourcePermissionsBatch"),
			zap.String("userID", userID.String()),
			zap.Int("resources", len(resources)),
			zap.Error(err),
		)
		return nil, err
	}

	for _, resource := range resources {
		if _, seen := details[resource]; seen || !e.registered(resource) {
			continue
		}
		details[resource] = e.resourceDetail(load.perms, resource)
	}

	return details, nil
}

func (e *engine) registered(resource string) bool {
	if _, ok := e.registry.Get(resource); ok {
		return true
	}
	_, ok := e.registry.Get(e.registry.GetEffectiveResource(resource))
	return ok
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
			auditmetrics.PermissionPathEffective, time.Since(start),
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

func (e *engine) checkAPIKeyPermission(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
	start time.Time,
) (*permissionports.PermissionCheckResult, error) {
	perms, creatorID, err := e.computeAPIKeyPermissions(
		ctx,
		req.APIKeyID,
		req.BusinessUnitID,
		req.OrganizationID,
	)
	if err != nil {
		return nil, err
	}

	result := e.resolveAPIKeyPermission(perms, req, creatorID, start)
	if !result.Allowed {
		return result, nil
	}

	return e.applyAccessPolicies(
		ctx,
		req,
		result,
		e.newAccessPolicySource(req.OrganizationID, req.BusinessUnitID),
	)
}

func (e *engine) resolveAPIKeyPermission(
	perms *permissionports.CachedPermissions,
	req *permissionports.PermissionCheckRequest,
	creatorID pulid.ID,
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

	dataScope := permission.DataScope(resourcePerms.DataScope)
	asCreator := *req
	asCreator.UserID = creatorID
	if result := enforceResourceAttributes(&asCreator, dataScope); result != nil {
		result.CheckDuration = time.Since(start).Milliseconds()
		return result
	}

	return &permissionports.PermissionCheckResult{
		Allowed:       true,
		Reason:        "allowed",
		DataScope:     dataScope,
		CacheHit:      false,
		CheckDuration: time.Since(start).Milliseconds(),
	}
}

func (e *engine) computeAPIKeyPermissions(
	ctx context.Context,
	apiKeyID, buID, orgID pulid.ID,
) (*permissionports.CachedPermissions, pulid.ID, error) {
	key, err := e.apiKeyRepo.GetByID(ctx, pagination.TenantInfo{
		OrgID: orgID,
		BuID:  buID,
	}, apiKeyID)
	if err != nil {
		return nil, pulid.Nil, err
	}

	resources := make(map[string]*permissionports.CachedResourcePermission)
	for _, rp := range key.Permissions {
		e.mergeAPIKeyPermissionIntoCache(resources, rp)
	}

	creator, err := e.apiKeyCreatorPermissions(ctx, key.CreatedByID, buID, orgID)
	if err != nil {
		return nil, pulid.Nil, err
	}
	e.restrictToCreator(resources, creator)

	return &permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityRestricted),
		Resources:      resources,
		ExpiresAt:      timeutils.NowUnix() + int64(cacheTTL.Seconds()),
	}, key.CreatedByID, nil
}

func (e *engine) apiKeyCreatorPermissions(
	ctx context.Context,
	creatorID, buID, orgID pulid.ID,
) (*permissionports.CachedPermissions, error) {
	empty := &permissionports.CachedPermissions{
		Resources: make(map[string]*permissionports.CachedResourcePermission),
	}

	creator, err := e.userRepo.GetTenantMember(ctx, tenantports.GetTenantMemberRequest{
		UserID:     creatorID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		Now:        timeutils.NowUnix(),
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return empty, nil
		}
		return nil, err
	}
	if creator.ValidateStatus() != nil {
		return empty, nil
	}

	load, err := e.getOrComputePermissions(
		authctx.WithoutSessionRoleActivation(ctx),
		creator.ID,
		orgID,
	)
	if err != nil {
		return nil, err
	}

	return load.perms, nil
}

func (e *engine) restrictToCreator(
	resources map[string]*permissionports.CachedResourcePermission,
	creator *permissionports.CachedPermissions,
) {
	for resource, rp := range resources {
		kept := make([]string, 0, len(rp.Operations))
		for _, op := range rp.Operations {
			held, ok := e.resolveResourcePermission(creator, resource, permission.Operation(op))
			if !ok {
				continue
			}
			kept = append(kept, op)
			if permission.DataScope(rp.DataScope).IsMorePermissive(
				permission.DataScope(held.DataScope),
			) {
				rp.DataScope = held.DataScope
			}
		}

		if len(kept) == 0 {
			delete(resources, resource)
			continue
		}
		rp.Operations = kept
	}
}

func (e *engine) getOrComputePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (permissionLoadResult, error) {
	result := permissionLoadResult{}
	key := permissionports.PermissionCacheKey{
		UserID:  userID,
		OrgID:   orgID,
		Variant: permissionCacheVariant(ctx),
	}

	cacheStart := time.Now()
	cached, err := e.cacheRepo.Get(ctx, key)
	result.cacheLookupDuration = time.Since(cacheStart)

	outcome := auditmetrics.PermissionCacheMiss
	switch {
	case err != nil:
		outcome = auditmetrics.PermissionCacheError
		e.l.Warn("cache lookup failed, computing fresh", zap.Error(err))
	case cached != nil && cached.ExpiresAt > timeutils.NowUnix():
		outcome = auditmetrics.PermissionCacheHit
	}
	e.metrics.Permission.RecordCacheLookup(outcome, result.cacheLookupDuration)

	if outcome == auditmetrics.PermissionCacheHit {
		result.perms = cached
		result.cacheHit = true
		return result, nil
	}

	epoch, epochErr := e.cacheRepo.Epoch(ctx, userID, orgID)
	if epochErr != nil {
		e.l.Warn("failed to read the permission epoch, not caching", zap.Error(epochErr))
	}

	computeStart := time.Now()
	perms, err := e.computePermissions(ctx, userID, orgID)
	result.computeDuration = time.Since(computeStart)
	if err != nil {
		return result, err
	}
	e.metrics.Permission.RecordCompute(auditmetrics.PermissionPathCached, result.computeDuration)

	if epochErr == nil {
		perms.Epoch = epoch
		if cacheErr := e.cacheRepo.Set(ctx, key, perms, cacheTTL); cacheErr != nil {
			e.l.Warn("failed to cache permissions", zap.Error(cacheErr))
		}
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
			ExpiresAt:      permissionsExpireAt(assignments),
		}, nil
	}

	roleIDs := activePermissionRoleIDs(assignments, roleActivation, hasRoleActivation)

	if len(roleIDs) == 0 {
		return &permissionports.CachedPermissions{
			MaxSensitivity: string(permission.SensitivityPublic),
			Resources:      make(map[string]*permissionports.CachedResourcePermission),
			ExpiresAt:      permissionsExpireAt(assignments),
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
		ExpiresAt:      permissionsExpireAt(assignments),
	}, nil
}

func permissionsExpireAt(assignments []*rbac.UserRoleAssignment) int64 {
	expiresAt := timeutils.NowUnix() + int64(cacheTTL.Seconds())
	for _, assignment := range assignments {
		if assignment.ExpiresAt != nil && !assignment.IsExpired() {
			expiresAt = min(expiresAt, *assignment.ExpiresAt)
		}
	}

	return expiresAt
}

func activePermissionRoleIDs(
	assignments []*rbac.UserRoleAssignment,
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
	role *rbac.Role,
	maxSensitivity *permission.FieldSensitivity,
) {
	if role.MaxSensitivity.Level() > maxSensitivity.Level() {
		*maxSensitivity = role.MaxSensitivity
	}
}

func (e *engine) mergeRolePermissionsIntoCache(
	resources map[string]*permissionports.CachedResourcePermission,
	rolePermissions []*rbac.ResourcePermission,
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
			Operations: e.opsToStrings(permission.Implied(rp.Operations)),
			DataScope:  string(rp.DataScope),
		}
		return
	}

	existing.Operations = e.mergeOpStrings(existing.Operations, permission.Implied(rp.Operations))
	if rp.DataScope.IsMorePermissive(permission.DataScope(existing.DataScope)) {
		existing.DataScope = string(rp.DataScope)
	}
}

func (e *engine) mergeResourcePermissionIntoCache(
	resources map[string]*permissionports.CachedResourcePermission,
	rp *rbac.ResourcePermission,
) {
	existing, ok := resources[rp.Resource]
	if !ok {
		resources[rp.Resource] = &permissionports.CachedResourcePermission{
			Operations: e.opsToStrings(permission.Implied(rp.Operations)),
			DataScope:  string(rp.DataScope),
		}
		return
	}

	existing.Operations = e.mergeOpStrings(existing.Operations, permission.Implied(rp.Operations))
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
	roles []*rbac.Role,
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
					Operations: permission.Implied(rp.Operations),
					DataScope:  rp.DataScope,
					GrantedBy:  []string{role.Name},
				}
			} else {
				opSet := permission.NewOperationSet(existing.Operations...)
				opSet.Add(permission.Implied(rp.Operations)...)
				existing.Operations = permission.Normalize(opSet.ToSlice())

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
