package permission

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"sort"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	manifestVersion = "1.0"
	cacheTTL        = 30 * time.Minute
)

type Params struct {
	fx.In

	RoleRepository      repositories.RoleRepository
	APIKeyRepository    repositories.APIKeyRepository
	PermissionCacheRepo repositories.PermissionCacheRepository
	UserRepository      repositories.UserRepository
	Registry            *permission.Registry
	RouteRegistry       *permission.RouteRegistry
	Logger              *zap.Logger
}

type engine struct {
	roleRepo      repositories.RoleRepository
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

	isPlatformAdmin, err := e.userRepo.IsPlatformAdmin(ctx, req.UserID)
	if err != nil {
		log.Error("failed to check platform admin status", zap.Error(err))
		return nil, err
	}

	if isPlatformAdmin {
		return &services.PermissionCheckResult{
			Allowed:       true,
			Reason:        "platform_admin",
			DataScope:     permission.DataScopeAll,
			CacheHit:      false,
			CheckDuration: time.Since(start).Milliseconds(),
		}, nil
	}

	perms, cacheHit, err := e.getOrComputePermissions(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		log.Error("failed to get permissions", zap.Error(err))
		return nil, err
	}

	if perms.IsOrgAdmin {
		return &services.PermissionCheckResult{
			Allowed:       true,
			Reason:        "org_admin",
			DataScope:     permission.DataScopeOrganization,
			CacheHit:      cacheHit,
			CheckDuration: time.Since(start).Milliseconds(),
		}, nil
	}

	if perms.IsBusinessUnitAdmin {
		return &services.PermissionCheckResult{
			Allowed:       true,
			Reason:        "business_unit_admin",
			DataScope:     permission.DataScopeOrganization,
			CacheHit:      cacheHit,
			CheckDuration: time.Since(start).Milliseconds(),
		}, nil
	}

	effectiveResource := e.registry.GetEffectiveResource(req.Resource)

	resourcePerms, ok := perms.Resources[effectiveResource]
	if !ok {
		return &services.PermissionCheckResult{
			Allowed:       false,
			Reason:        "no_permission",
			DataScope:     "",
			CacheHit:      cacheHit,
			CheckDuration: time.Since(start).Milliseconds(),
		}, nil
	}

	if slices.Contains(resourcePerms.Operations, string(req.Operation)) {
		return &services.PermissionCheckResult{
			Allowed:       true,
			Reason:        "allowed",
			DataScope:     permission.DataScope(resourcePerms.DataScope),
			CacheHit:      cacheHit,
			CheckDuration: time.Since(start).Milliseconds(),
		}, nil
	}

	return &services.PermissionCheckResult{
		Allowed:       false,
		Reason:        "no_permission",
		DataScope:     "",
		CacheHit:      cacheHit,
		CheckDuration: time.Since(start).Milliseconds(),
	}, nil
}

func (e *engine) CheckBatch(
	ctx context.Context,
	req *services.BatchPermissionCheckRequest,
) (*services.BatchPermissionCheckResult, error) {
	start := time.Now()

	results := make([]services.PermissionCheckResult, len(req.Checks))

	for i, check := range req.Checks {
		result, err := e.Check(ctx, &services.PermissionCheckRequest{
			PrincipalType:  req.PrincipalType,
			PrincipalID:    req.PrincipalID,
			UserID:         req.UserID,
			APIKeyID:       req.APIKeyID,
			BusinessUnitID: req.BusinessUnitID,
			OrganizationID: req.OrganizationID,
			Resource:       check.Resource,
			Operation:      check.Operation,
			ResourceID:     check.ResourceID,
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

func (e *engine) GetLightManifest(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*services.LightPermissionManifest, error) {
	log := e.l.With(
		zap.String("operation", "GetLightManifest"),
		zap.String("userID", userID.String()),
		zap.String("orgID", orgID.String()),
	)

	isPlatformAdmin, err := e.userRepo.IsPlatformAdmin(ctx, userID)
	if err != nil {
		log.Error("failed to check platform admin status", zap.Error(err))
		return nil, err
	}

	orgSummaries, err := e.getUserOrgSummaries(ctx, userID)
	if err != nil {
		log.Error("failed to get user organizations", zap.Error(err))
		return nil, err
	}
	expiresAt := timeutils.NowUnix() + int64(cacheTTL.Seconds())

	if isPlatformAdmin {
		return e.newLightManifest(
			userID,
			orgID,
			true,
			true,
			true,
			permission.SensitivityConfidential,
			e.allResourceBitmasks(),
			orgSummaries,
			expiresAt,
		), nil
	}

	perms, _, err := e.getOrComputePermissions(ctx, userID, orgID)
	if err != nil {
		log.Error("failed to get permissions", zap.Error(err))
		return nil, err
	}

	permissions := e.lightResourceBitmasks(perms)
	return e.newLightManifest(
		userID,
		orgID,
		false,
		perms.IsOrgAdmin,
		perms.IsBusinessUnitAdmin,
		permission.FieldSensitivity(perms.MaxSensitivity),
		permissions,
		orgSummaries,
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

func (e *engine) allResourceBitmasks() map[string]uint32 {
	allPermissions := make(map[string]uint32)

	for _, def := range e.registry.All() {
		var bitmask uint32
		for _, opDef := range def.Operations {
			bitmask |= permission.OperationToBit[opDef.Operation]
		}
		allPermissions[def.Resource] = bitmask
	}

	return allPermissions
}

func (e *engine) lightResourceBitmasks(perms *repositories.CachedPermissions) map[string]uint32 {
	if perms.IsOrgAdmin || perms.IsBusinessUnitAdmin {
		return e.allResourceBitmasks()
	}

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
	isPlatformAdmin bool,
	isOrgAdmin bool,
	isBusinessUnitAdmin bool,
	maxSensitivity permission.FieldSensitivity,
	permissions map[string]uint32,
	orgSummaries []services.OrgSummary,
	expiresAt int64,
) *services.LightPermissionManifest {
	manifest := &services.LightPermissionManifest{
		Version:             manifestVersion,
		UserID:              userID,
		OrganizationID:      orgID,
		IsPlatformAdmin:     isPlatformAdmin,
		IsOrgAdmin:          isOrgAdmin,
		IsBusinessUnitAdmin: isBusinessUnitAdmin,
		MaxSensitivity:      maxSensitivity,
		Permissions:         permissions,
		RouteAccess:         e.computeRouteAccess(permissions),
		AvailableOrgs:       orgSummaries,
		ExpiresAt:           expiresAt,
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

	isPlatformAdmin, err := e.userRepo.IsPlatformAdmin(ctx, userID)
	if err != nil {
		log.Error("failed to check platform admin status", zap.Error(err))
		return nil, err
	}

	def, ok := e.registry.Get(resource)
	if !ok {
		def, ok = e.registry.Get(e.registry.GetEffectiveResource(resource))
		if !ok {
			return nil, nil //nolint:nilnil // this is expected for non-registered resources
		}
	}

	if isPlatformAdmin {
		ops := make([]permission.Operation, len(def.Operations))
		for i, opDef := range def.Operations {
			ops[i] = opDef.Operation
		}

		accessibleFields := e.getAccessibleFields(def, permission.SensitivityConfidential)

		return &services.ResourcePermissionDetail{
			Resource:         resource,
			Operations:       ops,
			DataScope:        permission.DataScopeAll,
			MaxSensitivity:   permission.SensitivityConfidential,
			AccessibleFields: accessibleFields,
		}, nil
	}

	perms, _, err := e.getOrComputePermissions(ctx, userID, orgID)
	if err != nil {
		log.Error("failed to get permissions", zap.Error(err))
		return nil, err
	}

	maxSensitivity := permission.FieldSensitivity(perms.MaxSensitivity)

	if perms.IsOrgAdmin || perms.IsBusinessUnitAdmin {
		ops := make([]permission.Operation, len(def.Operations))
		for i, opDef := range def.Operations {
			ops[i] = opDef.Operation
		}

		accessibleFields := e.getAccessibleFields(def, maxSensitivity)

		return &services.ResourcePermissionDetail{
			Resource:         resource,
			Operations:       ops,
			DataScope:        permission.DataScopeOrganization,
			MaxSensitivity:   maxSensitivity,
			AccessibleFields: accessibleFields,
		}, nil
	}

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
		IsOrgAdmin:          false,
		IsBusinessUnitAdmin: false,
		MaxSensitivity:      string(permission.SensitivityRestricted),
		Resources:           resources,
		ExpiresAt:           timeutils.NowUnix() + int64(cacheTTL.Seconds()),
	}, nil
}

func (e *engine) getOrComputePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*repositories.CachedPermissions, bool, error) {
	cached, err := e.cacheRepo.Get(ctx, userID, orgID)
	if err != nil {
		e.l.Warn("cache lookup failed, computing fresh", zap.Error(err))
	}

	if cached != nil && cached.ExpiresAt > timeutils.NowUnix() {
		return cached, true, nil
	}

	perms, err := e.computePermissions(ctx, userID, orgID)
	if err != nil {
		return nil, false, err
	}

	if cacheErr := e.cacheRepo.Set(ctx, userID, orgID, perms, cacheTTL); cacheErr != nil {
		e.l.Warn("failed to cache permissions", zap.Error(cacheErr))
	}

	return perms, false, nil
}

func (e *engine) computePermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*repositories.CachedPermissions, error) {
	assignments, err := e.roleRepo.GetUserRoleAssignments(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	roleIDs := make([]pulid.ID, 0, len(assignments))
	for _, a := range assignments {
		if !a.IsExpired() {
			roleIDs = append(roleIDs, a.RoleID)
		}
	}

	if len(roleIDs) == 0 {
		return &repositories.CachedPermissions{
			IsOrgAdmin:          false,
			IsBusinessUnitAdmin: false,
			MaxSensitivity:      string(permission.SensitivityPublic),
			Resources:           make(map[string]*repositories.CachedResourcePermission),
			ExpiresAt:           timeutils.NowUnix() + int64(cacheTTL.Seconds()),
		}, nil
	}

	roles, err := e.roleRepo.GetRolesWithInheritance(ctx, roleIDs)
	if err != nil {
		return nil, err
	}

	isOrgAdmin := false
	isBusinessUnitAdmin := false
	maxSensitivity := permission.SensitivityPublic
	resources := make(map[string]*repositories.CachedResourcePermission)

	for _, role := range roles {
		e.applyRoleMeta(role, &isOrgAdmin, &isBusinessUnitAdmin, &maxSensitivity)
		e.mergeRolePermissionsIntoCache(resources, role.Permissions)
	}

	hasBUAdminAccess, err := e.roleRepo.HasBusinessUnitAdminAccess(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}
	if hasBUAdminAccess {
		isBusinessUnitAdmin = true
	}

	return &repositories.CachedPermissions{
		IsOrgAdmin:          isOrgAdmin,
		IsBusinessUnitAdmin: isBusinessUnitAdmin,
		MaxSensitivity:      string(maxSensitivity),
		Resources:           resources,
		ExpiresAt:           timeutils.NowUnix() + int64(cacheTTL.Seconds()),
	}, nil
}

func (e *engine) applyRoleMeta(
	role *permission.Role,
	isOrgAdmin *bool,
	isBusinessUnitAdmin *bool,
	maxSensitivity *permission.FieldSensitivity,
) {
	if role.IsOrgAdmin {
		*isOrgAdmin = true
	}
	if role.IsBusinessUnitAdmin {
		*isBusinessUnitAdmin = true
	}

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
			ID:                  role.ID,
			Name:                role.Name,
			Description:         role.Description,
			IsSystem:            role.IsSystem,
			IsOrgAdmin:          role.IsOrgAdmin,
			IsBusinessUnitAdmin: role.IsBusinessUnitAdmin,
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
		IsPlatformAdmin     bool
		IsOrgAdmin          bool
		IsBusinessUnitAdmin bool
		MaxSensitivity      string
		Permissions         map[string]uint32
	}{
		IsPlatformAdmin:     manifest.IsPlatformAdmin,
		IsOrgAdmin:          manifest.IsOrgAdmin,
		IsBusinessUnitAdmin: manifest.IsBusinessUnitAdmin,
		MaxSensitivity:      string(manifest.MaxSensitivity),
		Permissions:         manifest.Permissions,
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
