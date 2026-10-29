package reportjobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/reporting/render"
	"github.com/emoss08/trenova/internal/infrastructure/storage/uploadpipe"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/reportjobs/reportcontract"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const heartbeatEveryRows = 5000

// rowsSidecarSuffix names the sidecar after the artifact it accompanies, so
// the two share a prefix and the expiry sweep finds both from one run row.
const rowsSidecarSuffix = ".rows.json"

type ActivitiesParams struct {
	fx.In

	RunRepo             reportports.ReportRunRepository
	DefinitionRepo      reportports.ReportDefinitionRepository
	CatalogVersions     reportports.ReportCatalogVersionRepository
	ScheduleRepo        reportports.ReportScheduleRepository
	WorkflowStarter     kernelports.WorkflowStarter
	OrganizationRepo    tenantports.OrganizationRepository
	UserRepo            tenantports.UserRepository
	Reporting           *reporting.Service
	Engine              reportports.ReportEngine
	Renderers           reportports.ReportRendererRegistry
	Storage             kernelports.Client
	NotificationService *notificationservice.Service
	EmailService        emailports.EmailService `optional:"true"`
	Templates           documenttemplateports.DocumentTemplateResolver
	RealtimeService     realtimeports.RealtimeService `optional:"true"`
	ReadScopes          permissionports.ReadScopes
	AuditService        auditports.AuditService
	Metrics             *metrics.Registry `optional:"true"`
	ReportingConfig     *reportingconfig.Config
	Logger              *zap.Logger
}

type Activities struct {
	runRepo      reportports.ReportRunRepository
	defRepo      reportports.ReportDefinitionRepository
	versions     reportports.ReportCatalogVersionRepository
	scheduleRepo reportports.ReportScheduleRepository
	workflows    kernelports.WorkflowStarter
	orgRepo      tenantports.OrganizationRepository
	userRepo     tenantports.UserRepository
	reporting    *reporting.Service
	engine       reportports.ReportEngine
	renderers    reportports.ReportRendererRegistry
	storage      kernelports.Client
	notification *notificationservice.Service
	email        emailports.EmailService
	templates    documenttemplateports.DocumentTemplateResolver
	realtime     realtimeports.RealtimeService
	readScopes   permissionports.ReadScopes
	audit        auditports.AuditService
	metrics      *metrics.Report
	canned       cannedLibrary
	cfg          *reportingconfig.Config
	l            *zap.Logger
}

//nolint:gocritic // fx.In parameter structs must be passed by value
func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		runRepo:      p.RunRepo,
		defRepo:      p.DefinitionRepo,
		versions:     p.CatalogVersions,
		scheduleRepo: p.ScheduleRepo,
		workflows:    p.WorkflowStarter,
		orgRepo:      p.OrganizationRepo,
		userRepo:     p.UserRepo,
		reporting:    p.Reporting,
		engine:       p.Engine,
		renderers:    p.Renderers,
		storage:      p.Storage,
		notification: p.NotificationService,
		templates:    p.Templates,
		email:        p.EmailService,
		realtime:     p.RealtimeService,
		readScopes:   p.ReadScopes,
		audit:        p.AuditService,
		metrics:      reportMetrics(p.Metrics),
		canned:       p.Reporting,
		cfg:          p.ReportingConfig,
		l:            p.Logger.Named("reportjobs"),
	}
}

func reportMetrics(registry *metrics.Registry) *metrics.Report {
	if registry == nil {
		return nil
	}
	return registry.Report
}

type cannedLibrary interface {
	GetCanned(key string) (*canned.Entry, error)
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
	payload *reportcontract.RunReportPayload,
) (*PreparedRun, error) {
	tenant := pagination.TenantInfo{
		OrgID: payload.OrganizationID,
		BuID:  payload.BusinessUnitID,
	}

	run, err := a.runRepo.GetByID(ctx, &reportports.GetReportRunRequest{
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

	org, err := a.orgRepo.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
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

	if err = a.reporting.Check(ctx, &reporting.PrepareParams{
		Tenant:     runnerTenant,
		Definition: definition,
		Params:     run.Params,
		Mode:       reportaccess.ModeExport,
		Timezone:   org.Timezone,
	}); err != nil {
		return nil, classifyCompileError(err)
	}

	requestedBy, locale, title, description := a.runDisplayMetadata(ctx, run, runnerTenant)

	return &PreparedRun{
		RunID:          run.ID,
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
		RequestedByID:  run.RequestedByID,
		RevisionID:     run.RevisionID,
		ScheduleID:     run.ScheduleID,
		CannedKey:      run.CannedKey,
		Format:         run.Format,
		Title:          title,
		Description:    description,
		Params:         run.Params,
		OrgTimezone:    org.Timezone,
		RequestedBy:    requestedBy,
		Locale:         locale,
		MaxRunSeconds:  int64(a.cfg.GetMaxRunDuration().Seconds()),
		WantDigest:     a.scheduleWantsDigest(ctx, runnerTenant, run.ScheduleID),
	}, nil
}

// scheduleWantsDigest asks whether this run feeds a schedule that needs rows
// retained — either to render an inline table or to evaluate a measure alert.
// Capturing is cheap but not free, so an ad-hoc run and a schedule that only
// links to the artifact never pay for it.
func (a *Activities) scheduleWantsDigest(
	ctx context.Context,
	tenant pagination.TenantInfo,
	scheduleID pulid.ID,
) bool {
	if scheduleID.IsNil() {
		return false
	}
	schedule, err := a.scheduleRepo.GetByID(ctx, &reportports.GetReportScheduleRequest{
		TenantInfo: tenant,
		ScheduleID: scheduleID,
	})
	if err != nil {
		// A digest is a presentation nicety; failing the run over it would be
		// the wrong trade.
		a.l.Warn("could not resolve schedule for report digest",
			zap.String("scheduleId", scheduleID.String()), zap.Error(err))
		return false
	}
	// A measure alert reads the grand total from the same capture, so it has
	// to opt in too — otherwise the condition has nothing to compare against
	// and would never fire.
	return schedule.Delivery.WantsInlineDigest() || schedule.Alert.TargetsMeasure()
}

func (a *Activities) resolveDefinition(
	ctx context.Context,
	tenant pagination.TenantInfo,
	revisionID pulid.ID,
	cannedKey string,
) (*ir.Definition, error) {
	if !revisionID.IsNil() {
		revision, err := a.defRepo.GetRevision(ctx, &reportports.GetReportRevisionRequest{
			TenantInfo: tenant,
			RevisionID: revisionID,
		})
		if err != nil {
			return nil, err
		}
		return revision.Definition, nil
	}

	entry, err := a.canned.GetCanned(cannedKey)
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("canned report %q is not registered in this build", cannedKey),
			ErrTypeReportValidation, err,
		)
	}
	return entry.Definition, nil
}

func withRunLocale(ctx context.Context, prepared *PreparedRun) context.Context {
	if locale, ok := i18n.Parse(prepared.Locale); ok {
		return i18n.WithLocale(ctx, locale)
	}
	return ctx
}

func (a *Activities) runDisplayMetadata(
	ctx context.Context,
	run *report.ReportRun,
	runnerTenant pagination.TenantInfo,
) (requestedBy, locale, title, description string) {
	if user, userErr := a.userRepo.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:   runnerTenant,
		LookupUserID: run.RequestedByID,
	}); userErr == nil {
		requestedBy = user.Name
		if parsed, ok := i18n.Parse(user.Locale); ok {
			locale = parsed.String()
		}
	}

	title, description = a.reportTitle(ctx, run)

	return requestedBy, locale, title, description
}

func (a *Activities) reportTitle(
	ctx context.Context,
	run *report.ReportRun,
) (title, description string) {
	title = "Report"

	switch {
	case !run.DefinitionID.IsNil():
		if def, defErr := a.defRepo.GetByID(ctx, &reportports.GetReportDefinitionRequest{
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
		if entry, err := a.canned.GetCanned(run.CannedKey); err == nil {
			title = entry.Name
			description = entry.Description
		}
	}

	return title, description
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

	ctx, err = a.withReadScope(ctx, tenant)
	if err != nil {
		return nil, err
	}
	ctx = withRunLocale(ctx, prepared)

	run, err := a.reporting.Prepare(ctx, &reporting.PrepareParams{
		Tenant:     tenant,
		Definition: definition,
		Params:     prepared.Params,
		Mode:       reportaccess.ModeExport,
		Timezone:   prepared.OrgTimezone,
	})
	if err != nil {
		return nil, classifyCompileError(err)
	}

	renderer, err := a.renderers.For(prepared.Format)
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			err.Error(), ErrTypeReportValidation, err,
		)
	}

	stream, err := a.engine.Run(run.Context, &run.Request)
	if err != nil {
		return nil, classifyEngineError(reporting.EngineError(run.Context, run.Access, err))
	}
	defer stream.Close()

	digest := newDigestCapture(reportports.MaxDigestRows)
	source := newRunReader(stream, func() {
		activity.RecordHeartbeat(ctx)
	}, heartbeatEveryRows, digest)

	attempt := activity.GetInfo(ctx).Attempt
	artifactKey := fmt.Sprintf("%s/%s/%s/%d/report.%s",
		a.cfg.GetArtifactPrefix(),
		prepared.OrganizationID,
		prepared.RunID,
		attempt,
		prepared.Format.Extension(),
	)

	result, err := a.renderToStorage(
		run.Context,
		prepared,
		definition,
		renderer,
		source,
		artifactKey,
	)
	if err != nil {
		return nil, classifyEngineError(reporting.EngineError(run.Context, run.Access, err))
	}
	result.CacheHit = stream.Cached()

	captured, digestErr := digest.digest(run.Context, stream.Columns())
	if digestErr != nil {
		a.l.Warn("failed to capture report digest", zap.Error(digestErr))

		return result, nil
	}
	if prepared.WantDigest {
		result.Digest = captured
	}
	result.Sample = runSampleOf(captured)

	return result, nil
}

func (a *Activities) withReadScope(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (context.Context, error) {
	if a.readScopes == nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"report runs cannot be narrowed to what their requester may read",
			ErrTypeReportAuthorization, nil,
		)
	}
	principal := permissionports.UserActor(tenant).PermissionCheck("", "")

	return readscope.With(ctx, a.readScopes.Resolver(principal)), nil
}

func (a *Activities) renderToStorage(
	ctx context.Context,
	prepared *PreparedRun,
	definition *ir.Definition,
	renderer reportports.ReportRenderer,
	source *runReader,
	artifactKey string,
) (*ExecuteResult, error) {
	meta := reportports.ReportRunMeta{
		Title:           prepared.Title,
		Description:     prepared.Description,
		GeneratedAtUnix: timeutils.NowUnix(),
		Timezone:        prepared.OrgTimezone,
		Locale:          prepared.Locale,
		RequestedBy:     prepared.RequestedBy,
		Params:          reporting.OrderedParams(definition, prepared.Params),
	}
	objectMetadata := map[string]string{
		"organization-id": prepared.OrganizationID.String(),
		"run-id":          prepared.RunID.String(),
	}
	uploadCap := a.cfg.GetMaxArtifactBytes() + artifactSlackBytes

	artifact := uploadpipe.Open(ctx, a.storage, uploadpipe.Params{
		Key:         artifactKey,
		ContentType: prepared.Format.ContentType(),
		Metadata:    objectMetadata,
		MaxBytes:    uploadCap,
	})

	rowsKey := artifactKey + rowsSidecarSuffix
	rowsUpload := uploadpipe.Open(ctx, a.storage, uploadpipe.Params{
		Key:         rowsKey,
		ContentType: report.FormatJSON.ContentType(),
		Metadata:    objectMetadata,
		MaxBytes:    uploadCap,
	})
	sidecarDone := make(chan error, 1)
	go func() {
		defer close(source.sideExit)
		_, err := export.JSON(ctx, rowsUpload.Writer(), source.Sidecar(), render.Options(
			ctx, &meta, a.cfg.GetMaxArtifactBytes(),
		))
		sidecarDone <- err
	}()

	stats, renderErr := renderer.Render(ctx, &reportports.ReportRenderRequest{
		Dataset: source,
		Sink:    artifact.Writer(),
		Meta:    meta,
	})
	if renderErr == nil {
		renderErr = drain(ctx, source)
	}
	source.Stop()
	sidecarErr := <-sidecarDone
	if renderErr != nil {
		rowsUpload.Abort(renderErr)
		artifact.Abort(renderErr)

		return nil, renderFailure(renderErr, a.cfg.GetMaxArtifactBytes())
	}

	rowsKey = a.closeRowsSidecar(ctx, sidecarErr, rowsUpload, rowsKey)

	uploadedBytes, err := artifact.Close()
	if err != nil {
		return nil, fmt.Errorf("upload report artifact: %w", err)
	}

	return &ExecuteResult{
		ArtifactKey: artifactKey,
		RowsKey:     rowsKey,
		RowCount:    stats.Rows,
		ByteSize:    uploadedBytes,
		Truncated:   stats.Truncated,
	}, nil
}

func runSampleOf(digest *reportports.ReportDigest) []byte {
	if digest.IsEmpty() || len(digest.Envelope) > report.MaxRunSampleBytes {
		return nil
	}

	return digest.Envelope
}

func drain(ctx context.Context, source *runReader) error {
	for {
		_, err := source.Next(ctx)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (a *Activities) closeRowsSidecar(
	ctx context.Context,
	err error,
	upload *uploadpipe.Pipe,
	rowsKey string,
) string {
	if err == nil {
		_, err = upload.Close()
	} else {
		upload.Abort(err)
	}
	if err == nil {
		return rowsKey
	}

	a.l.Warn("failed to write report rows sidecar; the run will not be comparable",
		zap.String("key", rowsKey), zap.Error(err))

	if delErr := a.storage.Delete(ctx, rowsKey); delErr != nil {
		a.l.Debug("failed to clean up a partial rows sidecar",
			zap.String("key", rowsKey), zap.Error(delErr))
	}

	return ""
}

func (a *Activities) FinalizeRunActivity(
	ctx context.Context,
	payload *FinalizePayload,
) error {
	tenant := pagination.TenantInfo{
		OrgID: payload.OrganizationID,
		BuID:  payload.BusinessUnitID,
	}

	run, err := a.runRepo.GetByID(ctx, &reportports.GetReportRunRequest{
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
		run.RowsKey = payload.RowsKey
		run.Sample = payload.Sample
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
	if err := a.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceReport,
		ResourceID:     run.ID.String(),
		Operation:      permission.OpExport,
		UserID:         run.RequestedByID,
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
		CurrentState: map[string]any{
			dataKeyStatus:    string(run.Status),
			dataKeyFormat:    string(run.Format),
			dataKeyRowCount:  run.RowCount,
			"byteSize":       run.ByteSize,
			dataKeyTruncated: run.Truncated,
			"trigger":        string(run.Trigger),
		},
	}); err != nil {
		a.l.Warn("failed to audit report run completion",
			zap.String("runId", run.ID.String()), zap.Error(err))
	}
}

func (a *Activities) emitNotification(ctx context.Context, run *report.ReportRun) {
	name, _ := a.reportTitle(ctx, run)

	kind := documenttemplate.KindNotificationReportRunCompleted
	priority := notification.PriorityMedium
	eventType := "report_run_completed"

	//nolint:exhaustive // only terminal failure states change the wording
	switch run.Status {
	case report.RunStatusFailed:
		kind = documenttemplate.KindNotificationReportRunFailed
		priority = notification.PriorityHigh
		eventType = "report_run_failed"
	case report.RunStatusCanceled:
		kind = documenttemplate.KindNotificationReportRunCanceled
		eventType = "report_run_canceled"
	default:
	}

	tenantInfo := pagination.TenantInfo{
		OrgID:  run.OrganizationID,
		BuID:   run.BusinessUnitID,
		UserID: run.RequestedByID,
	}
	wording, ok := a.renderNotification(
		ctx, tenantInfo, kind, run.ID, runNotificationContext(run, name),
	)
	if !ok {
		return
	}

	data := map[string]any{
		dataKeyRunID:      run.ID.String(),
		dataKeyStatus:     string(run.Status),
		dataKeyFormat:     string(run.Format),
		dataKeyRowCount:   run.RowCount,
		dataKeyTruncated:  run.Truncated,
		dataKeyReportName: name,
		dataKeyByteSize:   run.ByteSize,
	}
	if run.ArtifactExpiresAt > 0 {
		data[dataKeyArtifactExpiresAt] = run.ArtifactExpiresAt
	}
	if !run.ScheduleID.IsNil() {
		data[dataKeyScheduleID] = run.ScheduleID.String()
	}

	if _, err := a.notification.Create(ctx, &notification.Notification{
		OrganizationID: run.OrganizationID,
		BusinessUnitID: &run.BusinessUnitID,
		TargetUserID:   &run.RequestedByID,
		Channel:        notification.ChannelUser,
		EventType:      eventType,
		Priority:       priority,
		Title:          wording.Title,
		Message:        wording.Message,
		Data:           data,
		Source:         "reportjobs.FinalizeRun",
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
		&realtimeports.PublishResourceInvalidationRequest{
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

func runScope(ctx context.Context, run *report.ReportRun) context.Context {
	return dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
	})
}

func (a *Activities) CleanupExpiredArtifactsActivity(
	ctx context.Context,
) (*CleanupExpiredResult, error) {
	result := &CleanupExpiredResult{}
	now := timeutils.NowUnix()

	for {
		expired, err := a.runRepo.ListExpired(ctx, &reportports.ListExpiredReportRunsRequest{
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

			// The sidecar expires with the artifact it describes: a run whose
			// rows outlived its download would keep data around under a
			// retention nobody set. It does not count towards
			// DeletedArtifacts, whose unit is the run's artifact — the key is
			// left on the row so a read refuses with "expired" rather than
			// with "this run predates the sidecar".
			if run.RowsKey != "" {
				if err = a.storage.Delete(ctx, run.RowsKey); err != nil {
					a.l.Warn("failed to delete expired report rows sidecar",
						zap.String("runId", run.ID.String()),
						zap.String("key", run.RowsKey),
						zap.Error(err))
					continue
				}
			}

			run.Status = report.RunStatusExpired
			run.Sample = nil
			if _, err = a.runRepo.Update(runScope(ctx, run), run); err != nil {
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

	stale, err := a.runRepo.ListStale(ctx, &reportports.ListStaleReportRunsRequest{
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
		if _, err = a.runRepo.Update(runScope(ctx, run), run); err != nil {
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

var errArtifactTooLarge = uploadpipe.ErrTooLarge

const artifactSlackBytes = 1 << 20

func classifyEngineError(err error) error {
	var rateErr *errortypes.RateLimitError
	if errors.As(err, &rateErr) {
		return temporal.NewApplicationErrorWithOptions(rateErr.Error(), ErrTypeReportBusy,
			temporal.ApplicationErrorOptions{Cause: err, NextRetryDelay: busyRetryDelay})
	}
	var bizErr *errortypes.BusinessError
	if errors.As(err, &bizErr) {
		return temporal.NewNonRetryableApplicationError(
			bizErr.Error(),
			ErrTypeReportTooExpensive,
			err,
		)
	}
	return classifyCompileError(err)
}

func renderFailure(err error, maxArtifactBytes int64) error {
	var rowLimit *render.RowLimitError
	switch {
	case errors.As(err, &rowLimit):
		return temporal.NewNonRetryableApplicationError(
			rowLimit.Error(), ErrTypeReportTooExpensive, err,
		)
	case errors.Is(err, errArtifactTooLarge):
		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf(
				"report artifact exceeds the maximum size of %d bytes — narrow your filters or reduce columns",
				maxArtifactBytes,
			),
			ErrTypeReportTooExpensive, err,
		)
	default:
		return err
	}
}
