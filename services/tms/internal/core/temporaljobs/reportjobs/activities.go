package reportjobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	reportingcompiler "github.com/emoss08/trenova/internal/core/services/reporting/compiler"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const heartbeatEveryRows = 5000

type ActivitiesParams struct {
	fx.In

	RunRepo             repositories.ReportRunRepository
	DefinitionRepo      repositories.ReportDefinitionRepository
	ScheduleRepo        repositories.ReportScheduleRepository
	WorkflowStarter     services.WorkflowStarter
	OrganizationRepo    repositories.OrganizationRepository
	UserRepo            repositories.UserRepository
	Compiler            services.ReportCompiler
	Executor            services.ReportDatasetExecutor
	Renderers           services.ReportRendererRegistry
	ReportingDB         *postgres.ReportingConnection
	Storage             storage.Client
	NotificationService *notificationservice.Service
	RealtimeService     services.RealtimeService   `optional:"true"`
	ResultCache         services.ReportResultCache `optional:"true"`
	AuditService        services.AuditService
	Metrics             *metrics.Registry `optional:"true"`
	Config              *config.Config
	Logger              *zap.Logger
}

type Activities struct {
	runRepo      repositories.ReportRunRepository
	defRepo      repositories.ReportDefinitionRepository
	scheduleRepo repositories.ReportScheduleRepository
	workflows    services.WorkflowStarter
	orgRepo      repositories.OrganizationRepository
	userRepo     repositories.UserRepository
	compiler     services.ReportCompiler
	executor     services.ReportDatasetExecutor
	renderers    services.ReportRendererRegistry
	reportingDB  *postgres.ReportingConnection
	storage      storage.Client
	notification *notificationservice.Service
	realtime     services.RealtimeService
	resultCache  services.ReportResultCache
	audit        services.AuditService
	metrics      *metrics.Report
	canned       *canned.Registry
	cfg          *config.ReportingConfig
	l            *zap.Logger
}

//nolint:gocritic // fx.In parameter structs must be passed by value
func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		runRepo:      p.RunRepo,
		defRepo:      p.DefinitionRepo,
		scheduleRepo: p.ScheduleRepo,
		workflows:    p.WorkflowStarter,
		orgRepo:      p.OrganizationRepo,
		userRepo:     p.UserRepo,
		compiler:     p.Compiler,
		executor:     p.Executor,
		renderers:    p.Renderers,
		reportingDB:  p.ReportingDB,
		storage:      p.Storage,
		notification: p.NotificationService,
		realtime:     p.RealtimeService,
		resultCache:  p.ResultCache,
		audit:        p.AuditService,
		metrics:      reportMetrics(p.Metrics),
		canned:       canned.Default(),
		cfg:          p.Config.GetReportingConfig(),
		l:            p.Logger.Named("reportjobs"),
	}
}

func reportMetrics(registry *metrics.Registry) *metrics.Report {
	if registry == nil {
		return nil
	}
	return registry.Report
}

func classifyCompileError(err error) error {
	var multiErr *errortypes.MultiError
	if errors.As(err, &multiErr) {
		return temporal.NewNonRetryableApplicationError(
			multiErr.Error(), ErrTypeReportValidation, err,
		)
	}
	var authzErr *errortypes.AuthorizationError
	if errors.As(err, &authzErr) {
		return temporal.NewNonRetryableApplicationError(
			authzErr.Error(), ErrTypeReportAuthorization, err,
		)
	}
	return err
}

func (a *Activities) PrepareRunActivity(
	ctx context.Context,
	payload *RunReportPayload,
) (*PreparedRun, error) {
	tenant := pagination.TenantInfo{
		OrgID: payload.OrganizationID,
		BuID:  payload.BusinessUnitID,
	}

	run, err := a.runRepo.GetByID(ctx, &repositories.GetReportRunRequest{
		TenantInfo: tenant,
		RunID:      payload.RunID,
	})
	if err != nil {
		return nil, err
	}

	if run.Status.IsTerminal() {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("run is already in terminal state %q", run.Status),
			ErrTypeReportValidation, nil,
		)
	}
	if run.RevisionID.IsNil() && run.CannedKey == "" {
		return nil, temporal.NewNonRetryableApplicationError(
			"run has neither a definition revision nor a canned report bound",
			ErrTypeReportValidation, nil,
		)
	}

	info := activity.GetInfo(ctx)
	run.Status = report.RunStatusRunning
	run.StartedAt = timeutils.NowUnix()
	run.TemporalWorkflowID = info.WorkflowExecution.ID
	run.TemporalRunID = info.WorkflowExecution.RunID
	if run, err = a.runRepo.Update(ctx, run); err != nil {
		return nil, err
	}

	definition, err := a.resolveDefinition(ctx, tenant, run.RevisionID, run.CannedKey)
	if err != nil {
		return nil, err
	}

	org, err := a.orgRepo.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  payload.OrganizationID,
			BuID:   payload.BusinessUnitID,
			UserID: run.RequestedByID,
		},
	})
	if err != nil {
		return nil, err
	}

	runnerTenant := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: run.RequestedByID,
	}

	if _, err = a.compiler.ValidateAndAuthorize(ctx, &services.ReportCompileRequest{
		Definition:  definition,
		Tenant:      runnerTenant,
		Params:      run.Params,
		OrgTimezone: org.Timezone,
		NowUnix:     timeutils.NowUnix(),
	}); err != nil {
		return nil, classifyCompileError(err)
	}

	requestedBy, title, description := a.runDisplayMetadata(ctx, run, runnerTenant)

	return &PreparedRun{
		RunID:          run.ID,
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
		RequestedByID:  run.RequestedByID,
		RevisionID:     run.RevisionID,
		CannedKey:      run.CannedKey,
		Format:         run.Format,
		Title:          title,
		Description:    description,
		Params:         run.Params,
		OrgTimezone:    org.Timezone,
		RequestedBy:    requestedBy,
		MaxRunSeconds:  int64(a.cfg.GetMaxRunDuration().Seconds()),
	}, nil
}

func (a *Activities) resolveDefinition(
	ctx context.Context,
	tenant pagination.TenantInfo,
	revisionID pulid.ID,
	cannedKey string,
) (*report.Definition, error) {
	if !revisionID.IsNil() {
		revision, err := a.defRepo.GetRevision(ctx, &repositories.GetReportRevisionRequest{
			TenantInfo: tenant,
			RevisionID: revisionID,
		})
		if err != nil {
			return nil, err
		}
		return revision.Definition, nil
	}

	entry, ok := a.canned.Get(cannedKey)
	if !ok {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("canned report %q is not registered in this build", cannedKey),
			ErrTypeReportValidation, nil,
		)
	}
	return entry.Definition, nil
}

func (a *Activities) runDisplayMetadata(
	ctx context.Context,
	run *report.ReportRun,
	runnerTenant pagination.TenantInfo,
) (requestedBy, title, description string) {
	title = "Report"

	if user, userErr := a.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:   runnerTenant,
		LookupUserID: run.RequestedByID,
	}); userErr == nil {
		requestedBy = user.Name
	}

	switch {
	case !run.DefinitionID.IsNil():
		if def, defErr := a.defRepo.GetByID(ctx, &repositories.GetReportDefinitionRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: run.OrganizationID,
				BuID:  run.BusinessUnitID,
			},
			DefinitionID: run.DefinitionID,
		}); defErr == nil {
			title = def.Name
			description = def.Description
		}
	case run.CannedKey != "":
		if entry, ok := a.canned.Get(run.CannedKey); ok {
			title = entry.Name
			description = entry.Description
		}
	}

	return requestedBy, title, description
}

func (a *Activities) ExecuteAndRenderActivity(
	ctx context.Context,
	prepared *PreparedRun,
) (*ExecuteResult, error) {
	tenant := pagination.TenantInfo{
		OrgID:  prepared.OrganizationID,
		BuID:   prepared.BusinessUnitID,
		UserID: prepared.RequestedByID,
	}

	definition, err := a.resolveDefinition(ctx, tenant, prepared.RevisionID, prepared.CannedKey)
	if err != nil {
		return nil, err
	}

	compiled, err := a.compiler.Compile(ctx, &services.ReportCompileRequest{
		Definition:  definition,
		Tenant:      tenant,
		Params:      prepared.Params,
		OrgTimezone: prepared.OrgTimezone,
		NowUnix:     timeutils.NowUnix(),
	})
	if err != nil {
		return nil, classifyCompileError(err)
	}

	if err = reportingcompiler.PreflightCost(ctx, a.reportingDB.DB(), compiled,
		reportingcompiler.CostLimits{
			MaxEstimatedCost: a.cfg.GetExplainCostLimit(),
			MaxEstimatedRows: a.cfg.GetExplainRowLimit(),
		},
	); err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			err.Error(), ErrTypeReportTooExpensive, err,
		)
	}

	cached, cacheKey := a.cachedResult(ctx, compiled, prepared)
	if cached != nil {
		return cached, nil
	}

	maxRows := a.cfg.GetMaxRows()
	if prepared.Format == report.FormatPDF && a.cfg.GetPDFMaxRows() < maxRows {
		maxRows = a.cfg.GetPDFMaxRows() + 1
	}

	reader, err := a.executor.Open(ctx, &services.OpenReportDatasetRequest{
		Compiled: compiled,
		MaxRows:  maxRows,
		Timeout:  a.cfg.GetStatementTimeout(),
	})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	heartbeating := &heartbeatingReader{inner: reader, ctx: ctx}

	renderer, err := a.renderers.For(prepared.Format)
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			err.Error(), ErrTypeReportValidation, err,
		)
	}

	attempt := activity.GetInfo(ctx).Attempt
	artifactKey := fmt.Sprintf("%s/%s/%s/%d/report.%s",
		a.cfg.GetArtifactPrefix(),
		prepared.OrganizationID,
		prepared.RunID,
		attempt,
		prepared.Format.Extension(),
	)

	result, err := a.renderToStorage(ctx, prepared, renderer, heartbeating, artifactKey)
	if err != nil {
		return nil, err
	}

	if a.resultCache != nil && cacheKey != "" {
		expiresAt := timeutils.NowUnix() + int64(a.cfg.GetArtifactRetention().Seconds())
		result.ArtifactExpiresAt = expiresAt
		if storeErr := a.resultCache.Store(ctx, cacheKey, &services.ReportCacheEntry{
			ArtifactKey:       result.ArtifactKey,
			RowCount:          result.RowCount,
			ByteSize:          result.ByteSize,
			Truncated:         result.Truncated,
			ArtifactExpiresAt: expiresAt,
		}); storeErr != nil {
			a.l.Warn("failed to store report cache entry", zap.Error(storeErr))
		}
	}

	return result, nil
}

// cachedResult returns a previously cached artifact for an identical compiled
// query (same SQL, args, format, and data-version vector), plus the cache key
// to store a fresh result under on a miss.
func (a *Activities) cachedResult(
	ctx context.Context,
	compiled *services.CompiledReportQuery,
	prepared *PreparedRun,
) (cached *ExecuteResult, cacheKey string) {
	if a.resultCache == nil {
		return nil, ""
	}

	cacheKey, err := a.resultCache.Key(
		ctx,
		compiled,
		prepared.Format,
		prepared.OrganizationID,
	)
	if err != nil {
		a.l.Warn("failed to derive report cache key", zap.Error(err))
		return nil, ""
	}

	entry, hit, err := a.resultCache.Lookup(ctx, cacheKey)
	if err != nil {
		a.l.Warn("report cache lookup failed", zap.Error(err))
		return nil, cacheKey
	}
	if !hit {
		if a.metrics != nil {
			a.metrics.RecordCacheLookup("miss")
		}
		return nil, cacheKey
	}

	if a.metrics != nil {
		a.metrics.RecordCacheLookup("hit")
	}
	return &ExecuteResult{
		ArtifactKey:       entry.ArtifactKey,
		RowCount:          entry.RowCount,
		ByteSize:          entry.ByteSize,
		Truncated:         entry.Truncated,
		CacheHit:          true,
		ArtifactExpiresAt: entry.ArtifactExpiresAt,
	}, cacheKey
}

func (a *Activities) renderToStorage(
	ctx context.Context,
	prepared *PreparedRun,
	renderer services.ReportRenderer,
	dataset services.ReportDatasetReader,
	artifactKey string,
) (*ExecuteResult, error) {
	pipeReader, pipeWriter := io.Pipe()
	uploadDone := make(chan error, 1)
	var uploadedBytes int64

	go func() {
		info, uploadErr := a.storage.Upload(ctx, &storage.UploadParams{
			Key:         artifactKey,
			ContentType: prepared.Format.ContentType(),
			Size:        -1,
			Body:        pipeReader,
			Metadata: map[string]string{
				"organization-id": prepared.OrganizationID.String(),
				"run-id":          prepared.RunID.String(),
			},
		})
		if uploadErr != nil {
			pipeReader.CloseWithError(uploadErr)
			uploadDone <- uploadErr
			return
		}
		uploadedBytes = info.Size
		uploadDone <- nil
	}()

	limited := &limitWriter{inner: pipeWriter, remaining: a.cfg.GetMaxArtifactBytes()}
	stats, renderErr := renderer.Render(ctx, &services.ReportRenderRequest{
		Dataset: dataset,
		Sink:    limited,
		Meta: services.ReportRunMeta{
			Title:           prepared.Title,
			Description:     prepared.Description,
			GeneratedAtUnix: timeutils.NowUnix(),
			Timezone:        prepared.OrgTimezone,
			RequestedBy:     prepared.RequestedBy,
			Params:          prepared.Params,
		},
	})

	if renderErr != nil {
		_ = pipeWriter.CloseWithError(renderErr)
		<-uploadDone
		if errors.Is(renderErr, errArtifactTooLarge) {
			return nil, temporal.NewNonRetryableApplicationError(
				fmt.Sprintf(
					"report artifact exceeds the maximum size of %d bytes — narrow your filters or reduce columns",
					a.cfg.GetMaxArtifactBytes(),
				),
				ErrTypeReportTooExpensive, renderErr,
			)
		}
		return nil, renderErr
	}

	if err := pipeWriter.Close(); err != nil {
		<-uploadDone
		return nil, err
	}
	if err := <-uploadDone; err != nil {
		return nil, fmt.Errorf("upload report artifact: %w", err)
	}

	return &ExecuteResult{
		ArtifactKey: artifactKey,
		RowCount:    stats.Rows,
		ByteSize:    uploadedBytes,
		Truncated:   stats.Truncated,
	}, nil
}

func (a *Activities) FinalizeRunActivity(
	ctx context.Context,
	payload *FinalizePayload,
) error {
	tenant := pagination.TenantInfo{
		OrgID: payload.OrganizationID,
		BuID:  payload.BusinessUnitID,
	}

	run, err := a.runRepo.GetByID(ctx, &repositories.GetReportRunRequest{
		TenantInfo: tenant,
		RunID:      payload.RunID,
	})
	if err != nil {
		return err
	}

	if run.Status.IsTerminal() {
		return nil
	}

	now := timeutils.NowUnix()
	run.Status = payload.Status
	run.Error = payload.Error
	run.RowCount = payload.RowCount
	run.ByteSize = payload.ByteSize
	run.Truncated = payload.Truncated
	run.DurationMs = payload.DurationMs
	run.CompletedAt = now
	run.CacheHit = payload.CacheHit
	if payload.Status == report.RunStatusSucceeded {
		run.ArtifactKey = payload.ArtifactKey
		run.ArtifactExpiresAt = now + int64(a.cfg.GetArtifactRetention().Seconds())
		if payload.ArtifactExpiresAt > 0 {
			run.ArtifactExpiresAt = payload.ArtifactExpiresAt
		}
	}

	if run, err = a.runRepo.Update(ctx, run); err != nil {
		return err
	}

	a.emitAudit(run)
	a.emitNotification(ctx, run)
	a.emitInvalidation(ctx, run)
	a.recordScheduleRunOutcome(ctx, run)

	if a.metrics != nil {
		a.metrics.RecordRun(
			string(run.Status), string(run.Format), string(run.Trigger),
			time.Duration(run.DurationMs)*time.Millisecond,
			run.RowCount, run.ByteSize,
		)
	}

	return nil
}

func (a *Activities) emitAudit(run *report.ReportRun) {
	if err := a.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceReport,
		ResourceID:     run.ID.String(),
		Operation:      permission.OpExport,
		UserID:         run.RequestedByID,
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
		CurrentState: map[string]any{
			"status":    string(run.Status),
			"format":    string(run.Format),
			"rowCount":  run.RowCount,
			"byteSize":  run.ByteSize,
			"truncated": run.Truncated,
			"trigger":   string(run.Trigger),
		},
	}); err != nil {
		a.l.Warn("failed to audit report run completion",
			zap.String("runId", run.ID.String()), zap.Error(err))
	}
}

func (a *Activities) emitNotification(ctx context.Context, run *report.ReportRun) {
	title := "Your report is ready"
	message := "The report finished generating and is ready to download."
	priority := notification.PriorityMedium
	eventType := "report_run_completed"

	//nolint:exhaustive // only terminal failure states change the message
	switch run.Status {
	case report.RunStatusFailed:
		title = "Report generation failed"
		message = "The report could not be generated."
		if run.Error != nil && run.Error.Message != "" {
			message = run.Error.Message
		}
		priority = notification.PriorityHigh
		eventType = "report_run_failed"
	case report.RunStatusCanceled:
		title = "Report run canceled"
		message = "The report run was canceled."
		eventType = "report_run_canceled"
	default:
	}

	if _, err := a.notification.Create(ctx, &notification.Notification{
		OrganizationID: run.OrganizationID,
		BusinessUnitID: &run.BusinessUnitID,
		TargetUserID:   &run.RequestedByID,
		Channel:        notification.ChannelUser,
		EventType:      eventType,
		Priority:       priority,
		Title:          title,
		Message:        message,
		Data: map[string]any{
			"runId":     run.ID.String(),
			"status":    string(run.Status),
			"format":    string(run.Format),
			"rowCount":  run.RowCount,
			"truncated": run.Truncated,
		},
		Source: "reportjobs.FinalizeRun",
	}); err != nil {
		a.l.Warn("failed to create report run notification",
			zap.String("runId", run.ID.String()), zap.Error(err))
	}
}

func (a *Activities) emitInvalidation(ctx context.Context, run *report.ReportRun) {
	if a.realtime == nil {
		return
	}
	if err := a.realtime.PublishResourceInvalidation(
		ctx,
		&services.PublishResourceInvalidationRequest{
			OrganizationID: run.OrganizationID,
			BusinessUnitID: run.BusinessUnitID,
			Resource:       "report-run",
			Action:         "updated",
			RecordID:       run.ID,
			EntityVersion:  run.Version,
			ActorUserID:    run.RequestedByID,
		},
	); err != nil {
		a.l.Warn("failed to publish report run invalidation",
			zap.String("runId", run.ID.String()), zap.Error(err))
	}
}

func (a *Activities) CleanupExpiredArtifactsActivity(
	ctx context.Context,
) (*CleanupExpiredResult, error) {
	result := &CleanupExpiredResult{}
	now := timeutils.NowUnix()

	for {
		expired, err := a.runRepo.ListExpired(ctx, &repositories.ListExpiredReportRunsRequest{
			CutoffUnix: now,
			Limit:      100,
		})
		if err != nil {
			return nil, err
		}
		if len(expired) == 0 {
			if a.metrics != nil {
				a.metrics.RecordArtifactsCleaned(result.DeletedArtifacts)
			}
			return result, nil
		}

		for _, run := range expired {
			activity.RecordHeartbeat(ctx, run.ID.String())

			if run.ArtifactKey != "" {
				if err = a.storage.Delete(ctx, run.ArtifactKey); err != nil {
					a.l.Warn("failed to delete expired report artifact",
						zap.String("runId", run.ID.String()),
						zap.String("key", run.ArtifactKey),
						zap.Error(err))
					continue
				}
				result.DeletedArtifacts++
			}

			run.Status = report.RunStatusExpired
			if _, err = a.runRepo.Update(ctx, run); err != nil {
				a.l.Warn("failed to mark report run expired",
					zap.String("runId", run.ID.String()), zap.Error(err))
				continue
			}
			result.ExpiredRuns++
		}
	}
}

func (a *Activities) ReconcileZombieRunsActivity(
	ctx context.Context,
) (*ReconcileZombiesResult, error) {
	result := &ReconcileZombiesResult{}
	staleCutoff := timeutils.NowUnix() -
		int64((a.cfg.GetMaxRunDuration() + time.Hour).Seconds())

	stale, err := a.runRepo.ListStale(ctx, &repositories.ListStaleReportRunsRequest{
		Statuses:          []report.RunStatus{report.RunStatusQueued, report.RunStatusRunning},
		UpdatedBeforeUnix: staleCutoff,
		Limit:             200,
	})
	if err != nil {
		return nil, err
	}

	for _, run := range stale {
		activity.RecordHeartbeat(ctx, run.ID.String())

		run.Status = report.RunStatusFailed
		run.Error = &report.RunError{
			Code:    "ZOMBIE",
			Message: "The run was abandoned without completing and has been marked failed",
		}
		run.CompletedAt = timeutils.NowUnix()
		if _, err = a.runRepo.Update(ctx, run); err != nil {
			a.l.Warn("failed to fail zombie report run",
				zap.String("runId", run.ID.String()), zap.Error(err))
			continue
		}
		result.ZombieRuns++
	}

	if a.metrics != nil {
		a.metrics.RecordZombieRunsReconciled(result.ZombieRuns)
	}

	return result, nil
}

var errArtifactTooLarge = errors.New("report artifact exceeds the maximum size")

type limitWriter struct {
	inner     io.Writer
	remaining int64
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, errArtifactTooLarge
	}
	n, err := w.inner.Write(p)
	w.remaining -= int64(n)
	return n, err
}

type heartbeatingReader struct {
	inner services.ReportDatasetReader
	ctx   context.Context
	rows  int64
}

func (r *heartbeatingReader) Schema() []services.ReportResultColumn { return r.inner.Schema() }

func (r *heartbeatingReader) Next(ctx context.Context) (services.ReportRow, error) {
	row, err := r.inner.Next(ctx)
	if err == nil {
		r.rows++
		if r.rows%heartbeatEveryRows == 0 {
			activity.RecordHeartbeat(r.ctx, r.rows)
		}
	}
	return row, err
}

func (r *heartbeatingReader) RowCount() int64 { return r.inner.RowCount() }

func (r *heartbeatingReader) Truncated() bool { return r.inner.Truncated() }

func (r *heartbeatingReader) Close() error { return r.inner.Close() }
