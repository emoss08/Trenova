package permissionservice

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/sourcegraph/conc"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// CacheWorkerService handles background caching of permission data
type CacheWorkerService interface {
	QueueCacheJob(userID, organizationID pulid.ID, manifest *ports.PermissionManifest)
}

type cacheJob struct {
	userID         pulid.ID
	organizationID pulid.ID
	manifest       *ports.PermissionManifest
	retryCount     int
}

type cacheWorkerService struct {
	cache     ports.PermissionCacheRepository
	logger    *zap.Logger
	cacheJobs chan cacheJob
	workers   int
	workerWg  *conc.WaitGroup
	shutdown  chan struct{}
	once      sync.Once
}

type CacheWorkerParams struct {
	fx.In

	Cache  ports.PermissionCacheRepository
	Config *config.Config
	Logger *zap.Logger
}

func NewCacheWorkerService(p CacheWorkerParams, lc fx.Lifecycle) CacheWorkerService {
	cfg := p.Config.PermissionCache
	service := &cacheWorkerService{
		cache:    p.Cache,
		logger:   p.Logger.Named("permission-cache-worker"),
		workers:  cfg.Workers,
		workerWg: conc.NewWaitGroup(),
		shutdown: make(chan struct{}),
	}

	service.cacheJobs = make(chan cacheJob, cfg.BufferSize)
	lc.Append(fx.Hook{
		OnStart: service.start,
		OnStop:  service.stop,
	})

	return service
}

func (s *cacheWorkerService) start(_ context.Context) error {
	s.logger.Info("starting permission cache worker service",
		zap.Int("workers", s.workers),
		zap.Int("bufferSize", cap(s.cacheJobs)))

	for i := 0; i < s.workers; i++ {
		workerID := pulid.MustNew("perm_worker_")
		s.workerWg.Go(func() {
			s.workerLoop(workerID)
		})
	}

	return nil
}

func (s *cacheWorkerService) stop(ctx context.Context) error {
	s.logger.Info("stopping permission cache worker service")

	s.once.Do(func() {
		close(s.shutdown)
	})

	done := make(chan struct{})
	go func() {
		s.workerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("permission cache worker service stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("permission cache worker service shutdown timed out")
		return ctx.Err()
	}

	return nil
}

func (s *cacheWorkerService) QueueCacheJob(
	userID, organizationID pulid.ID,
	manifest *ports.PermissionManifest,
) {
	job := cacheJob{
		userID:         userID,
		organizationID: organizationID,
		manifest:       manifest,
		retryCount:     0,
	}

	select {
	case s.cacheJobs <- job:
		s.logger.Debug("queued cache job",
			zap.String("userID", userID.String()),
			zap.String("orgID", organizationID.String()))
	case <-s.shutdown:
		s.logger.Debug("dropped cache job - service shutting down")
	default:
		s.logger.Warn("failed to queue cache job - channel full")
	}
}

func (s *cacheWorkerService) workerLoop(workerID pulid.ID) {
	log := s.logger.With(zap.String("workerID", workerID.String()))
	log.Debug("worker started")

	defer log.Debug("worker stopped")

	for {
		select {
		case job := <-s.cacheJobs:
			s.processCacheJob(log, job)
		case <-s.shutdown:
			log.Debug("worker received shutdown signal")
			return
		}
	}
}

func (s *cacheWorkerService) processCacheJob(log *zap.Logger, job cacheJob) {
	const maxRetries = 3
	const baseDelay = time.Second

	log = log.With(
		zap.String("userID", job.userID.String()),
		zap.String("orgID", job.organizationID.String()),
		zap.Int("retryCount", job.retryCount),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := s.cachePermissionsSync(ctx, job.userID, job.organizationID, job.manifest)
	if err != nil {
		log.Warn("failed to cache permissions", zap.Error(err))

		if job.retryCount < maxRetries {
			job.retryCount++
			delay := baseDelay * time.Duration(job.retryCount)

			log.Debug("retrying cache job",
				zap.Int("retryCount", job.retryCount),
				zap.Duration("delay", delay))

			go func() {
				time.Sleep(delay)
				select {
				case s.cacheJobs <- job:
				case <-s.shutdown:
				default:
					log.Error("failed to requeue cache job - channel full")
				}
			}()
		} else {
			log.Error("cache job failed after max retries", zap.Error(err))
		}
	} else {
		log.Debug("successfully cached permissions")
	}
}

func (s *cacheWorkerService) cachePermissionsSync(
	ctx context.Context,
	userID, organizationID pulid.ID,
	manifest *ports.PermissionManifest,
) error {
	dataScopes := make(map[string]permission.DataScope)

	cached := &ports.CachedPermissions{
		Version:    manifest.Version,
		ComputedAt: manifest.ComputedAt,
		ExpiresAt:  manifest.ExpiresAt,
		Permissions: &ports.CompiledPermissions{
			Resources:   manifest.Resources,
			GlobalFlags: 0,
			DataScopes:  dataScopes,
		},
		BloomFilter: manifest.BloomFilter,
		Checksum:    manifest.Checksum,
	}

	return s.cache.Set(ctx, userID, organizationID, cached, 30*time.Minute)
}
