package aiauditservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/fieldsensitivity"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/storage/uploadpipe"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	downloadURLExpiry   = 60 * time.Second
	exportKeyPrefix     = "ai-audit-exports/"
	staleExportAge      = 6 * time.Hour
	cleanupBatchSize    = 100
	exportNoticeSource  = "aiauditservice.Exports"
	exportErrorLimit    = 1000
	exportWorkflowIDTag = "ai-audit-export/"
	noticeKeyKind       = "kind"
	noticeKeyStatus     = "status"
)

// Notifier creates one person's notification.
type Notifier interface {
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
}

// Exports asks for, writes, hands out and expires trail exports.
type Exports struct {
	ledger      repositories.AIAuditRepository
	exports     repositories.AIAuditExportRepository
	source      repositories.AIAuditSourceRepository
	exporter    *Exporter
	storage     storage.Client
	workflows   serviceports.WorkflowStarter
	notifier    Notifier
	realtime    serviceports.RealtimeService
	audit       serviceports.AuditService
	permissions serviceports.PermissionEngine
	cfg         *config.AIAuditExportConfig
	metrics     *metrics.AIAudit
	now         func() time.Time
	l           *zap.Logger
}

type ExportsParams struct {
	Ledger      repositories.AIAuditRepository
	Exports     repositories.AIAuditExportRepository
	Source      repositories.AIAuditSourceRepository
	Exporter    *Exporter
	Storage     storage.Client
	Workflows   serviceports.WorkflowStarter
	Notifier    Notifier
	Realtime    serviceports.RealtimeService
	Audit       serviceports.AuditService
	Permissions serviceports.PermissionEngine
	Config      *config.AIAuditExportConfig
	Metrics     *metrics.AIAudit
	Now         func() time.Time
	Logger      *zap.Logger
}

func NewExports(p *ExportsParams) *Exports {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	cfg := p.Config
	if cfg == nil {
		cfg = &config.AIAuditExportConfig{}
	}

	return &Exports{
		ledger:      p.Ledger,
		exports:     p.Exports,
		source:      p.Source,
		exporter:    p.Exporter,
		storage:     p.Storage,
		workflows:   p.Workflows,
		notifier:    p.Notifier,
		realtime:    p.Realtime,
		audit:       p.Audit,
		permissions: p.Permissions,
		cfg:         cfg,
		metrics:     p.Metrics,
		now:         now,
		l:           p.Logger.Named("aiaudit.exports"),
	}
}

func scopeOf(export *aiaudit.AIAuditExport) *repositories.AIAuditEventScope {
	return &repositories.AIAuditEventScope{
		TenantInfo: pagination.TenantInfo{
			OrgID: export.OrganizationID,
			BuID:  export.BusinessUnitID,
		},
		From:        export.RangeFrom,
		To:          export.RangeTo,
		SnapshotSeq: export.SnapshotSeq,
		Filter:      export.Filters,
	}
}

// Request records an export and writes it: at once when it is small, else on
// the report queue. Either way the request is audited before anything is
// written.
func (x *Exports) Request(
	ctx context.Context,
	req *serviceports.RequestAIAuditExportRequest,
) (*aiaudit.AIAuditExport, error) {
	actor := req.Actor
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"An export of the AI audit trail must be requested by a person",
		)
	}

	filter := req.Filter
	if filter == nil {
		filter = &aiaudit.ExportFilter{}
	}
	export := &aiaudit.AIAuditExport{
		OrganizationID:    actor.OrganizationID,
		BusinessUnitID:    actor.BusinessUnitID,
		RequestedByUserID: actor.UserID,
		Format:            req.Format,
		Filters:           filter,
		RangeFrom:         req.From,
		RangeTo:           req.To,
		Status:            aiaudit.ExportStatusPending,
	}

	multiErr := errortypes.NewMultiError()
	export.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	tenantInfo := pagination.TenantInfo{OrgID: actor.OrganizationID, BuID: actor.BusinessUnitID}
	head, err := x.ledger.GetChainHead(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	export.SnapshotSeq = head.LastSeq

	summary, err := x.ledger.Summarize(ctx, scopeOf(export))
	if err != nil {
		return nil, err
	}
	if summary.Count > int64(x.cfg.GetMaxRows()) {
		return nil, errortypes.NewValidationError(
			"to",
			errortypes.ErrInvalid,
			"This export would hold {0} rows; one export can hold at most {1}. Narrow the range or the filters.",
			summary.Count,
			x.cfg.GetMaxRows(),
		)
	}

	created, err := x.exports.Create(ctx, export)
	if err != nil {
		return nil, err
	}
	x.auditExport(created, actor, "export_requested", summary.Count)

	if summary.Count <= int64(x.cfg.GetSyncMaxRows()) {
		return x.Run(ctx, tenantInfo, created.ID, nil)
	}

	return x.start(ctx, created)
}

func (x *Exports) start(
	ctx context.Context,
	export *aiaudit.AIAuditExport,
) (*aiaudit.AIAuditExport, error) {
	workflowID := exportWorkflowIDTag + export.ID.String()
	if _, err := x.workflows.StartWorkflow(ctx,
		client.StartWorkflowOptions{
			ID:                    workflowID,
			TaskQueue:             temporaltype.ReportTaskQueue,
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		},
		serviceports.AIAuditExportWorkflowName,
		&serviceports.AIAuditExportPayload{
			ExportID:       export.ID,
			OrganizationID: export.OrganizationID,
			BusinessUnitID: export.BusinessUnitID,
		},
	); err != nil {
		x.l.Error("failed to start an AI audit export", zap.Error(err))
		if _, failErr := x.fail(ctx, export, "The export could not be queued"); failErr != nil {
			x.l.Error("failed to mark an unqueued AI audit export failed", zap.Error(failErr))
		}

		return nil, errortypes.NewBusinessError(
			"The export could not be queued — try again shortly",
		)
	}

	export.Status = aiaudit.ExportStatusRunning
	export.WorkflowID = workflowID
	started := x.now().Unix()
	export.StartedAt = &started

	return x.exports.Update(ctx, export)
}

// Run writes an export's file. It is what the workflow's activity calls, and
// what a small export runs inline; an export already finished is returned as
// it is, so a retried activity never writes twice.
func (x *Exports) Run(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	exportID pulid.ID,
	heartbeat Heartbeat,
) (*aiaudit.AIAuditExport, error) {
	export, err := x.exports.GetByID(ctx, repositories.GetAIAuditExportRequest{
		TenantInfo: tenantInfo,
		ID:         exportID,
	})
	if err != nil {
		return nil, err
	}
	if export.Status.Terminal() {
		return export, nil
	}

	if export.Status != aiaudit.ExportStatusRunning || export.StartedAt == nil {
		started := x.now().Unix()
		export.Status = aiaudit.ExportStatusRunning
		export.StartedAt = &started
		if export, err = x.exports.Update(ctx, export); err != nil {
			return nil, err
		}
	}

	stats, key, runErr := x.write(ctx, export, heartbeat)
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) {
			return nil, runErr
		}
		x.l.Error("failed to write an AI audit export",
			zap.String("exportId", export.ID.String()), zap.Error(runErr))

		return x.fail(ctx, export, runErr.Error())
	}

	now := x.now().Unix()
	expires := now + int64(x.cfg.GetTTL().Seconds())
	export.Status = aiaudit.ExportStatusSucceeded
	export.RowCount = stats.Rows
	export.ByteSize = stats.Bytes
	export.SHA256 = stats.SHA256
	export.ArtifactKey = key
	export.ArtifactExpiresAt = &expires
	export.ChainKeyID = x.exporter.keyring.ActiveKeyID()
	if stats.Rows > 0 {
		first, last := stats.FirstSeq, stats.LastSeq
		export.ChainFirstSeq = &first
		export.ChainLastSeq = &last
	}
	export.ChainComplete = stats.Complete
	export.CompletedAt = &now
	export.ErrorMessage = ""

	updated, err := x.exports.Update(ctx, export)
	if err != nil {
		return nil, err
	}
	x.metrics.RecordExport(string(updated.Format), string(updated.Status))
	x.notify(ctx, updated)
	x.invalidate(ctx, updated)

	return updated, nil
}

func (x *Exports) write(
	ctx context.Context,
	export *aiaudit.AIAuditExport,
	heartbeat Heartbeat,
) (*ExportStats, string, error) {
	summary, err := x.ledger.Summarize(ctx, scopeOf(export))
	if err != nil {
		return nil, "", err
	}

	requester, err := x.source.UserNames(ctx,
		pagination.TenantInfo{OrgID: export.OrganizationID, BuID: export.BusinessUnitID},
		[]pulid.ID{export.RequestedByUserID},
	)
	if err != nil {
		return nil, "", err
	}

	key := exportKeyPrefix + export.OrganizationID.String() + "/" + export.ID.String() + "." +
		export.Format.Extension()
	pipe := uploadpipe.Open(ctx, x.storage, uploadpipe.Params{
		Key:         key,
		ContentType: export.Format.ContentType(),
		Metadata: map[string]string{
			"organization-id": export.OrganizationID.String(),
			"export-id":       export.ID.String(),
		},
	})

	stats, err := x.exporter.Write(ctx, pipe.Writer(), &ExportSpec{
		Export:  export,
		Summary: summary,
		Ceilings: fieldsensitivity.NewCeilings(
			x.permissions, export.RequestedByUserID, export.OrganizationID,
		),
		RequestedBy:      requester[export.RequestedByUserID],
		LinkAuditEntries: x.mayReadAuditLog(ctx, export),
		GeneratedAt:      x.now(),
		Heartbeat:        heartbeat,
	})
	if err != nil {
		pipe.Abort(err)

		return nil, "", err
	}

	if _, err = pipe.Close(); err != nil {
		return nil, "", fmt.Errorf("upload AI audit export: %w", err)
	}

	return stats, key, nil
}

func (x *Exports) mayReadAuditLog(ctx context.Context, export *aiaudit.AIAuditExport) bool {
	if x.permissions == nil {
		return false
	}

	result, err := x.permissions.Check(ctx, &serviceports.PermissionCheckRequest{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    export.RequestedByUserID,
		UserID:         export.RequestedByUserID,
		BusinessUnitID: export.BusinessUnitID,
		OrganizationID: export.OrganizationID,
		Resource:       permission.ResourceAuditLog.String(),
		Operation:      permission.OpRead,
	})

	return err == nil && result != nil && result.Allowed
}

func (x *Exports) fail(
	ctx context.Context,
	export *aiaudit.AIAuditExport,
	message string,
) (*aiaudit.AIAuditExport, error) {
	now := x.now().Unix()
	export.Status = aiaudit.ExportStatusFailed
	export.ErrorMessage = truncateRunes(message, exportErrorLimit)
	export.CompletedAt = &now

	updated, err := x.exports.Update(ctx, export)
	if err != nil {
		return nil, err
	}
	x.metrics.RecordExport(string(updated.Format), string(updated.Status))
	x.notify(ctx, updated)
	x.invalidate(ctx, updated)

	return updated, nil
}

// Download is a short-lived link to an export's file for the person who asked
// for it. The file was written at their ceiling, so nobody else is handed it.
func (x *Exports) Download(
	ctx context.Context,
	req *serviceports.GetAIAuditExportDownloadRequest,
) (*serviceports.AIAuditExportDownload, error) {
	actor := req.Actor
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Only the person who asked for an export may download it",
		)
	}

	export, err := x.exports.GetByID(ctx, repositories.GetAIAuditExportRequest{
		TenantInfo: pagination.TenantInfo{OrgID: actor.OrganizationID, BuID: actor.BusinessUnitID},
		ID:         req.ExportID,
	})
	if err != nil {
		return nil, err
	}
	if export.RequestedByUserID != actor.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the person who asked for an export may download it",
		)
	}

	now := x.now()
	if export.Status == aiaudit.ExportStatusExpired ||
		(export.ArtifactExpiresAt != nil && *export.ArtifactExpiresAt <= now.Unix()) {
		return nil, errortypes.NewBusinessError("This export has expired — export the trail again")
	}
	if !export.Downloadable(now.Unix()) {
		return nil, errortypes.NewBusinessError("This export has no file to download yet")
	}

	fileName := export.FileName()
	url, err := x.storage.GetPresignedURL(ctx, &storage.PresignedURLParams{
		Key:                export.ArtifactKey,
		Expiry:             downloadURLExpiry,
		ContentDisposition: fmt.Sprintf("attachment; filename=%q", fileName),
	})
	if err != nil {
		x.l.Error("failed to presign an AI audit export",
			zap.String("exportId", export.ID.String()), zap.Error(err))

		return nil, errortypes.NewBusinessError(
			"The export could not be prepared for download — try again shortly",
		)
	}

	x.auditExport(export, actor, "export_downloaded", export.RowCount)

	return &serviceports.AIAuditExportDownload{
		URL:       url,
		FileName:  fileName,
		ExpiresAt: now.Add(downloadURLExpiry).Unix(),
		SHA256:    export.SHA256,
	}, nil
}

func (x *Exports) auditExport(
	export *aiaudit.AIAuditExport,
	actor *serviceports.RequestActor,
	event string,
	rows int64,
) {
	if x.audit == nil {
		return
	}

	if err := x.audit.LogAction(&serviceports.LogActionParams{
		Resource:       permission.ResourceAIAuditTrail,
		ResourceID:     export.ID.String(),
		Operation:      permission.OpExport,
		UserID:         actor.UserID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		APIKeyID:       actor.APIKeyID,
		OrganizationID: export.OrganizationID,
		BusinessUnitID: export.BusinessUnitID,
		Critical:       true,
		CurrentState: map[string]any{
			"event":     event,
			"format":    string(export.Format),
			"rangeFrom": export.RangeFrom,
			"rangeTo":   export.RangeTo,
			"rowCount":  rows,
			"filtered":  !export.Filters.Unfiltered(),
		},
	}); err != nil {
		x.l.Error("failed to audit an AI audit export",
			zap.String("exportId", export.ID.String()),
			zap.String("event", event),
			zap.Error(err))
	}
}

func (x *Exports) notify(ctx context.Context, export *aiaudit.AIAuditExport) {
	if x.notifier == nil {
		return
	}

	eventType := serviceports.AIAuditExportReadyEvent
	priority := notification.PriorityMedium
	title := "Your AI audit trail export is ready"
	message := "Open the audit trail's exports in AI Control to download it."
	if export.Status == aiaudit.ExportStatusFailed {
		eventType = serviceports.AIAuditExportFailedEvent
		priority = notification.PriorityHigh
		title = "Your AI audit trail export failed"
		message = "The export could not be written. Open the audit trail's exports in AI Control to try again."
	}

	buID := export.BusinessUnitID
	userID := export.RequestedByUserID
	correlation := export.ID.String() + ":" + string(export.Status)
	data := map[string]any{
		noticeKeyKind:   eventType,
		"exportId":      export.ID.String(),
		noticeKeyStatus: string(export.Status),
		"format":        string(export.Format),
		"rowCount":      export.RowCount,
	}
	if path, ok := productguide.RecordPath(
		serviceports.AIAuditExportRecordEntity,
		export.ID.String(),
	); ok {
		data["link"] = path
	}

	if _, err := x.notifier.Create(ctx, &notification.Notification{
		OrganizationID:  export.OrganizationID,
		BusinessUnitID:  &buID,
		TargetUserID:    &userID,
		Channel:         notification.ChannelUser,
		EventType:       eventType,
		Priority:        priority,
		Title:           title,
		Message:         message,
		Source:          exportNoticeSource,
		CorrelationID:   &correlation,
		Data:            data,
		RelatedEntities: map[string]any{"aiAuditExportId": export.ID.String()},
	}); err != nil {
		x.l.Warn("failed to tell a person their AI audit export finished",
			zap.String("exportId", export.ID.String()), zap.Error(err))
	}
}

func (x *Exports) invalidate(ctx context.Context, export *aiaudit.AIAuditExport) {
	if x.realtime == nil {
		return
	}

	if err := x.realtime.PublishResourceInvalidation(ctx,
		&serviceports.PublishResourceInvalidationRequest{
			OrganizationID: export.OrganizationID,
			BusinessUnitID: export.BusinessUnitID,
			AudienceUserID: export.RequestedByUserID,
			Resource:       serviceports.AIAuditExportResource,
			Action:         "updated",
			RecordID:       export.ID,
			EntityVersion:  export.Version,
			ActorUserID:    export.RequestedByUserID,
		}); err != nil {
		x.l.Warn("failed to announce an AI audit export",
			zap.String("exportId", export.ID.String()), zap.Error(err))
	}
}

// CleanupResult is what an expiry sweep did.
type CleanupResult struct {
	Expired int `json:"expired"`
	Failed  int `json:"failed"`
}

// Cleanup deletes files past their download window and fails exports whose
// worker is gone.
func (x *Exports) Cleanup(ctx context.Context, heartbeat Heartbeat) (*CleanupResult, error) {
	result := &CleanupResult{}
	now := x.now()

	for {
		expired, err := x.exports.ListExpired(ctx, now.Unix(), cleanupBatchSize)
		if err != nil {
			return nil, err
		}
		if len(expired) == 0 {
			break
		}

		progressed := false
		for _, export := range expired {
			beat(heartbeat, export.ID.String())
			if err = x.storage.Delete(ctx, export.ArtifactKey); err != nil {
				x.l.Warn("failed to delete an expired AI audit export",
					zap.String("exportId", export.ID.String()), zap.Error(err))

				continue
			}
			export.Status = aiaudit.ExportStatusExpired
			export.ArtifactKey = ""
			if _, err = x.exports.Update(ctx, export); err != nil {
				x.l.Warn("failed to mark an AI audit export expired",
					zap.String("exportId", export.ID.String()), zap.Error(err))

				continue
			}
			progressed = true
			result.Expired++
			x.invalidate(ctx, export)
		}
		if !progressed || len(expired) < cleanupBatchSize {
			break
		}
	}

	stale, err := x.exports.ListStale(ctx, now.Add(-staleExportAge).Unix(), cleanupBatchSize)
	if err != nil {
		return nil, err
	}
	for _, export := range stale {
		beat(heartbeat, export.ID.String())
		if _, err = x.fail(ctx, export, "The export was abandoned before it finished"); err != nil {
			x.l.Warn("failed to fail an abandoned AI audit export",
				zap.String("exportId", export.ID.String()), zap.Error(err))

			continue
		}
		result.Failed++
	}

	return result, nil
}
