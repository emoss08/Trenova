package permissionservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/permissionregistry"
	"github.com/emoss08/trenova/pkg/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type EngineParams struct {
	fx.In

	Registry    *permissionregistry.Registry
	PolicyRepo  ports.PolicyRepository
	RoleRepo    ports.RoleRepository
	UserRepo    repositories.UserRepository
	Cache       ports.PermissionCacheRepository
	Compiler    ports.PolicyCompiler
	CacheWorker CacheWorkerService
	Logger      *zap.Logger
}

type permissionEngine struct {
	registry     *permissionregistry.Registry
	policyRepo   ports.PolicyRepository
	roleRepo     ports.RoleRepository
	userRepo     repositories.UserRepository
	cache        ports.PermissionCacheRepository
	compiler     ports.PolicyCompiler
	cacheWorker  CacheWorkerService
	logger       *zap.Logger
	cacheVersion string
}

//nolint:gocritic // dependencies injection
func NewPermissionEngine(p EngineParams) ports.PermissionEngine {
	return &permissionEngine{
		registry:     p.Registry,
		policyRepo:   p.PolicyRepo,
		roleRepo:     p.RoleRepo,
		userRepo:     p.UserRepo,
		cache:        p.Cache,
		compiler:     p.Compiler,
		cacheWorker:  p.CacheWorker,
		logger:       p.Logger.Named("permission-engine"),
		cacheVersion: "v3.0",
	}
}

func (e *permissionEngine) Check(
	ctx context.Context,
	req *ports.PermissionCheckRequest,
) (*ports.PermissionCheckResult, error) {
	start := time.Now()
	log := e.logger.With(
		zap.String("operation", "Check"),
		zap.String("userID", req.UserID.String()),
		zap.String("orgID", req.OrganizationID.String()),
		zap.String("resource", req.ResourceType),
		zap.String("action", req.Action),
	)

	// Check if user has admin role - admins can do everything
	isAdmin, err := e.HasAdminRole(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		log.Error("failed to check admin role", zap.Error(err))
		return nil, err
	}

	if isAdmin {
		result := &ports.PermissionCheckResult{
			Allowed:       true,
			Reason:        "user has admin role",
			CacheHit:      false,
			ComputeTimeMs: float64(time.Since(start).Microseconds()) / 1000.0,
		}

		log.Debug("permission check completed - admin user",
			zap.Bool("allowed", true),
			zap.Float64("timeMs", result.ComputeTimeMs),
		)

		return result, nil
	}

	manifest, cacheHit, err := e.getUserPermissionsInternal(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		log.Error("failed to get user permissions", zap.Error(err))
		return nil, err
	}

	allowed, reason := e.checkPermission(manifest, req)

	result := &ports.PermissionCheckResult{
		Allowed:       allowed,
		Reason:        reason,
		CacheHit:      cacheHit,
		ComputeTimeMs: float64(time.Since(start).Microseconds()) / 1000.0,
	}

	log.Debug("permission check completed",
		zap.Bool("allowed", allowed),
		zap.Bool("cacheHit", cacheHit),
		zap.Float64("timeMs", result.ComputeTimeMs),
	)

	return result, nil
}

func (e *permissionEngine) CheckBatch(
	ctx context.Context,
	req *ports.BatchPermissionCheckRequest,
) (*ports.BatchPermissionCheckResult, error) {
	start := time.Now()
	log := e.logger.With(
		zap.String("operation", "CheckBatch"),
		zap.String("userID", req.UserID.String()),
		zap.String("orgID", req.OrganizationID.String()),
		zap.Int("checkCount", len(req.Checks)),
	)

	isAdmin, err := e.HasAdminRole(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		log.Error("failed to check admin role", zap.Error(err))
		return nil, err
	}

	if isAdmin {
		results := make([]*ports.PermissionCheckResult, len(req.Checks))
		for i := range req.Checks {
			results[i] = &ports.PermissionCheckResult{
				Allowed:  true,
				Reason:   "user has admin role",
				CacheHit: false,
			}
		}

		totalTime := time.Since(start)
		result := &ports.BatchPermissionCheckResult{
			Results:      results,
			CacheHitRate: 0,
			TotalTimeMs:  float64(totalTime.Microseconds()) / 1000.0,
		}

		log.Debug("batch permission check completed - admin user",
			zap.Float64("totalTimeMs", result.TotalTimeMs),
		)

		return result, nil
	}

	manifest, cacheHit, err := e.getUserPermissionsInternal(ctx, req.UserID, req.OrganizationID)
	if err != nil {
		log.Error("failed to get user permissions", zap.Error(err))
		return nil, err
	}

	results := make([]*ports.PermissionCheckResult, len(req.Checks))
	cacheHitCount := 0
	if cacheHit {
		cacheHitCount = len(req.Checks)
	}

	for i, check := range req.Checks {
		allowed, reason := e.checkPermission(manifest, check)

		results[i] = &ports.PermissionCheckResult{
			Allowed:  allowed,
			Reason:   reason,
			CacheHit: cacheHit,
		}
	}

	totalTime := time.Since(start)
	result := &ports.BatchPermissionCheckResult{
		Results:      results,
		CacheHitRate: float64(cacheHitCount) / float64(len(req.Checks)),
		TotalTimeMs:  float64(totalTime.Microseconds()) / 1000.0,
	}

	log.Debug("batch permission check completed",
		zap.Float64("cacheHitRate", result.CacheHitRate),
		zap.Float64("totalTimeMs", result.TotalTimeMs),
	)

	return result, nil
}

func (e *permissionEngine) GetUserPermissions(
	ctx context.Context,
	userID, organizationID pulid.ID,
) (*ports.PermissionManifest, error) {
	log := e.logger.With(
		zap.String("operation", "GetUserPermissions"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
	)

	manifest, cacheHit, err := e.getUserPermissionsInternal(ctx, userID, organizationID)
	if err != nil {
		log.Error("failed to get user permissions", zap.Error(err))
		return nil, err
	}

	log.Debug("retrieved user permissions", zap.Bool("cacheHit", cacheHit))
	return manifest, nil
}

func (e *permissionEngine) RefreshUserPermissions(
	ctx context.Context,
	userID, organizationID pulid.ID,
) error {
	log := e.logger.With(
		zap.String("operation", "RefreshUserPermissions"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
	)

	if err := e.cache.Delete(ctx, userID, organizationID); err != nil {
		log.Warn("failed to delete cache", zap.Error(err))
		return err
	}

	_, _, err := e.getUserPermissionsInternal(ctx, userID, organizationID)
	if err != nil {
		log.Error("failed to refresh permissions", zap.Error(err))
		return err
	}

	log.Info("refreshed user permissions")
	return nil
}

func (e *permissionEngine) InvalidateCache(
	ctx context.Context,
	userID, organizationID pulid.ID,
) error {
	return e.cache.Delete(ctx, userID, organizationID)
}

func (e *permissionEngine) getUserPermissionsInternal(
	ctx context.Context,
	userID, organizationID pulid.ID,
) (*ports.PermissionManifest, bool, error) {
	cached, err := e.cache.Get(ctx, userID, organizationID)
	if err == nil && cached != nil && cached.ExpiresAt.After(time.Now()) {
		manifest := e.cachedToManifest(cached, userID, organizationID)
		return manifest, true, nil
	}

	manifest, err := e.computeUserPermissions(ctx, userID, organizationID)
	if err != nil {
		return nil, false, err
	}

	e.cacheWorker.QueueCacheJob(userID, organizationID, manifest)

	return manifest, false, nil
}

func (e *permissionEngine) computeUserPermissions(
	ctx context.Context,
	userID, organizationID pulid.ID,
) (*ports.PermissionManifest, error) {
	log := e.logger.With(
		zap.String("operation", "computeUserPermissions"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
	)

	policies, err := e.policyRepo.GetUserPolicies(ctx, userID, organizationID)
	if err != nil {
		log.Error("failed to get user policies", zap.Error(err))
		return nil, err
	}

	log.Debug("retrieved user policies", zap.Int("count", len(policies)))

	compiled, err := e.compiler.CompileForUser(userID, organizationID, policies)
	if err != nil {
		log.Error("failed to compile policies", zap.Error(err))
		return nil, err
	}

	bloomFilter, err := e.compiler.BuildBloomFilter(compiled)
	if err != nil {
		log.Error("failed to build bloom filter", zap.Error(err))
		return nil, err
	}

	checksum, err := e.computeChecksum(compiled)
	if err != nil {
		log.Error("failed to compute checksum", zap.Error(err))
		return nil, err
	}

	availableOrgs, err := e.getUserAvailableOrgs(ctx, userID)
	if err != nil {
		log.Warn("failed to get user organizations, using current org only", zap.Error(err))
		availableOrgs = []pulid.ID{organizationID}
	}

	now := time.Now()
	manifest := &ports.PermissionManifest{
		Version:       e.cacheVersion,
		UserID:        userID,
		CurrentOrg:    organizationID,
		AvailableOrgs: availableOrgs,
		ComputedAt:    now,
		ExpiresAt:     now.Add(30 * time.Minute),
		Resources:     compiled.Resources,
		BloomFilter:   bloomFilter,
		Checksum:      checksum,
	}

	log.Debug("computed user permissions",
		zap.Int("resourceCount", len(manifest.Resources)),
		zap.Int("bloomFilterSize", len(bloomFilter)),
	)

	return manifest, nil
}

func (e *permissionEngine) checkPermission(
	manifest *ports.PermissionManifest,
	req *ports.PermissionCheckRequest,
) (allowed bool, reason string) {
	if _, exists := e.registry.GetResource(req.ResourceType); !exists {
		e.logger.Warn("resource not found in registry",
			zap.String("resource", req.ResourceType),
		)
		return false, fmt.Sprintf(
			"resource %s not registered in permission registry",
			req.ResourceType,
		)
	}

	key := fmt.Sprintf("%s:%s", req.ResourceType, req.Action)
	if !e.testBloomFilter(manifest.BloomFilter, key) {
		return false, "permission not found in bloom filter"
	}

	resourcePerms, ok := manifest.Resources[req.ResourceType]
	if !ok {
		return false, fmt.Sprintf("no permissions for resource: %s", req.ResourceType)
	}

	if ports.HasAction(resourcePerms.StandardOps, req.Action) {
		return true, "allowed"
	}

	if slices.Contains(resourcePerms.ExtendedOps, req.Action) {
		return true, "allowed"
	}

	return false, fmt.Sprintf("action %s not allowed on resource %s", req.Action, req.ResourceType)
}

func (e *permissionEngine) testBloomFilter(filter []byte, item string) bool {
	if len(filter) == 0 {
		return true
	}

	bloom := &bloomFilter{
		bits:      filter,
		size:      len(filter),
		hashCount: 7,
	}

	return bloom.Test(item)
}

func (e *permissionEngine) cachedToManifest(
	cached *ports.CachedPermissions,
	userID, organizationID pulid.ID,
) *ports.PermissionManifest {
	return &ports.PermissionManifest{
		Version:       cached.Version,
		UserID:        userID,
		CurrentOrg:    organizationID,
		AvailableOrgs: []pulid.ID{organizationID},
		ComputedAt:    cached.ComputedAt,
		ExpiresAt:     cached.ExpiresAt,
		Resources:     cached.Permissions.Resources,
		BloomFilter:   cached.BloomFilter,
		Checksum:      cached.Checksum,
	}
}

func (e *permissionEngine) getUserAvailableOrgs(
	ctx context.Context,
	userID pulid.ID,
) ([]pulid.ID, error) {
	user, err := e.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
		UserID:       userID,
		IncludeOrgs:  true,
		IncludeRoles: false,
	})
	if err != nil {
		return nil, err
	}

	orgs := make([]pulid.ID, 0, len(user.OrganizationMemberships))
	for i := range user.OrganizationMemberships {
		orgs = append(orgs, user.OrganizationMemberships[i].OrganizationID)
	}

	return orgs, nil
}

func (e *permissionEngine) computeChecksum(data any) (string, error) {
	jsonData, err := sonic.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:]), nil
}

func (e *permissionEngine) HasAdminRole(
	ctx context.Context,
	userID, organizationID pulid.ID,
) (bool, error) {
	return e.roleRepo.HasAdminRole(ctx, userID, organizationID)
}
