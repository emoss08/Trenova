package auditexportservice

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config/auditconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/storage/uploadpipe"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	ExportWorkflowName = "AuditExportWorkflow"
	exportKeyPrefix    = "audit-exports/"
	exportPageSize     = 1000
	maxExportRows      = 2_000_000
	maxExportBytes     = 2 << 30
	downloadURLExpiry  = 60 * time.Second
	cleanupBatchSize   = 100
	noticeSource       = "auditexportservice"
)

type ExportPayload struct {
	ExportID       pulid.ID `json:"exportId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

type Params struct {
	fx.In

	Exports       auditports.ExportRepository
	Entries       auditports.AuditRepository
	Storage       kernelports.Client
	Workflows     kernelports.WorkflowStarter
	Notifications *notificationservice.Service `optional:"true"`
	Auditor       permissionports.SecurityAuditor
	Config        *auditconfig.Config
	Metrics       *metrics.Registry
	Logger        *zap.Logger
}

type Service struct {
	exports       auditports.ExportRepository
	entries       auditports.AuditRepository
	storage       kernelports.Client
	workflows     kernelports.WorkflowStarter
	notifications *notificationservice.Service
	auditor       permissionports.SecurityAuditor
	cfg           *auditconfig.Config
	metrics       *metrics.Registry
	now           func() time.Time
	l             *zap.Logger
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) *Service {
	return &Service{
		exports:       p.Exports,
		entries:       p.Entries,
		storage:       p.Storage,
		workflows:     p.Workflows,
		notifications: p.Notifications,
		auditor:       p.Auditor,
		cfg:           p.Config,
		metrics:       p.Metrics,
		now:           time.Now,
		l:             p.Logger.Named("service.audit-export"),
	}
}

type RequestExport struct {
	TenantInfo      pagination.TenantInfo
	Actor           kernelports.AuditActor
	Format          audit.ExportFormat
	RangeFrom       int64
	RangeTo         int64
	Filters         audit.ExportFilters
	Access          auditports.EntryAccess
	VisibleGroups   protectedfield.GroupSet
	IncludeSecurity bool
}

func (s *Service) Request(ctx context.Context, req *RequestExport) (*audit.Export, error) {
	filters := req.Filters
	filters.VisibleGroups = uint8(req.VisibleGroups)
	filters.Resources = narrowResources(filters.Resources, req.Access)
	if !req.Access.AllResources && len(filters.Resources) == 0 {
		return nil, errortypes.NewAuthorizationError(
			"You cannot read any of the kinds of record you asked to export",
		)
	}

	export := &audit.Export{
		ID:              pulid.MustNew("aex_"),
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		RequestedByID:   req.TenantInfo.UserID,
		Format:          req.Format,
		Filters:         filters,
		RangeFrom:       req.RangeFrom,
		RangeTo:         req.RangeTo,
		IncludeSecurity: req.IncludeSecurity && req.Access.IncludeSecurity,
		Status:          audit.ExportStatusPending,
	}
	multiErr := errortypes.NewMultiError()
	export.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.exports.Create(ctx, export)
	if err != nil {
		return nil, err
	}

	s.record(ctx, req.Actor, created, audit.ActionExportRequested, "Audit log export requested")

	workflowID := "audit-export/" + created.ID.String()
	if _, err = s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             temporaltype.AuditTaskQueue,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}, ExportWorkflowName, &ExportPayload{
		ExportID:       created.ID,
		OrganizationID: created.OrganizationID,
		BusinessUnitID: created.BusinessUnitID,
	}); err != nil {
		s.l.Error("failed to start an audit export", zap.Error(err))
		if _, failErr := s.fail(ctx, created, "The export could not be queued"); failErr != nil {
			s.l.Error("failed to mark an unqueued audit export failed", zap.Error(failErr))
		}
		return nil, errortypes.NewBusinessError("The export could not be queued — try again shortly")
	}

	created.WorkflowID = workflowID
	return s.exports.Update(ctx, created)
}

func narrowResources(requested []string, access auditports.EntryAccess) []string {
	if access.AllResources {
		return requested
	}
	readable := make(map[string]struct{}, len(access.Resources))
	for _, resource := range access.Resources {
		readable[resource.String()] = struct{}{}
	}
	if len(requested) == 0 {
		out := make([]string, 0, len(access.Resources))
		for _, resource := range access.Resources {
			out = append(out, resource.String())
		}
		return out
	}
	out := make([]string, 0, len(requested))
	for _, resource := range requested {
		if _, ok := readable[resource]; ok {
			out = append(out, resource)
		}
	}
	return out
}

func (s *Service) List(
	ctx context.Context,
	req *auditports.ListExportsRequest,
) (*pagination.CursorListResult[*audit.Export], error) {
	return s.exports.ListConnection(ctx, req)
}

func (s *Service) Run(
	ctx context.Context,
	tenant pagination.TenantInfo,
	exportID pulid.ID,
	heartbeat func(string),
) (*audit.Export, error) {
	export, err := s.exports.GetByID(ctx, auditports.GetExportRequest{ID: exportID, TenantInfo: tenant})
	if err != nil {
		return nil, err
	}
	if export.Status.Terminal() {
		return export, nil
	}

	started := s.now().Unix()
	export.Status = audit.ExportStatusRunning
	export.StartedAt = &started
	if export, err = s.exports.Update(ctx, export); err != nil {
		return nil, err
	}

	stats, key, runErr := s.write(ctx, tenant, export, heartbeat)
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) {
			return nil, runErr
		}
		s.l.Error("failed to write an audit export",
			zap.String("exportId", export.ID.String()), zap.Error(runErr))
		return s.fail(ctx, export, runErr.Error())
	}

	now := s.now().Unix()
	expires := now + int64(s.cfg.GetExportTTL().Seconds())
	export.Status = audit.ExportStatusSucceeded
	export.RowCount = stats.rows
	export.ByteSize = stats.bytes
	export.SHA256 = stats.sha256
	export.ArtifactKey = key
	export.ArtifactExpiresAt = &expires
	export.CompletedAt = &now
	export.ErrorMessage = ""

	updated, err := s.exports.Update(ctx, export)
	if err != nil {
		return nil, err
	}
	s.metrics.Audit.RecordExport(string(updated.Format), true)
	s.notify(ctx, updated)
	return updated, nil
}

type writeStats struct {
	rows   int64
	bytes  int64
	sha256 string
}

type countingWriter struct {
	w     io.Writer
	hash  hash.Hash
	bytes int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.bytes += int64(n)
	c.hash.Write(p[:n])
	return n, err
}

var csvHeader = []string{
	"id", "timestamp", "action", "resource", "resource_id", "operation", "category",
	"retention_class", "channel", "principal_type", "principal_id", "actor_name", "user_id",
	"api_key_id", "ai_run_id", "entity_ref", "comment", "ip_address", "user_agent",
	"correlation_id", "trace_id", "critical", "sensitive_data", "changes",
}

func (s *Service) write(
	ctx context.Context,
	tenant pagination.TenantInfo,
	export *audit.Export,
	heartbeat func(string),
) (*writeStats, string, error) {
	key := exportKeyPrefix + export.OrganizationID.String() + "/" + export.ID.String() + "." +
		export.Format.Extension()
	pipe := uploadpipe.Open(ctx, s.storage, uploadpipe.Params{
		Key:         key,
		ContentType: export.Format.ContentType(),
		MaxBytes:    maxExportBytes,
		Metadata: map[string]string{
			"organization-id": export.OrganizationID.String(),
			"export-id":       export.ID.String(),
		},
	})
	counter := &countingWriter{w: pipe.Writer(), hash: sha256.New()}

	stats, err := s.stream(ctx, tenant, export, counter, heartbeat)
	if err != nil {
		pipe.Abort(err)
		return nil, "", err
	}
	if _, err = pipe.Close(); err != nil {
		return nil, "", fmt.Errorf("upload audit export: %w", err)
	}

	stats.bytes = counter.bytes
	stats.sha256 = hex.EncodeToString(counter.hash.Sum(nil))
	return stats, key, nil
}

func (s *Service) stream(
	ctx context.Context,
	tenant pagination.TenantInfo,
	export *audit.Export,
	out io.Writer,
	heartbeat func(string),
) (*writeStats, error) {
	visible := protectedfield.GroupSet(export.Filters.VisibleGroups)
	var csvWriter *csv.Writer
	if export.Format == audit.ExportFormatCSV {
		csvWriter = csv.NewWriter(out)
		if err := csvWriter.Write(csvHeader); err != nil {
			return nil, err
		}
	}

	stats := &writeStats{}
	var afterTS int64
	var afterID pulid.ID
	for {
		page, err := s.entries.ExportPage(ctx, &auditports.ExportPageRequest{
			TenantInfo: tenant,
			Export:     export,
			AfterTS:    afterTS,
			AfterID:    afterID,
			Limit:      exportPageSize,
		})
		if err != nil {
			return nil, err
		}
		for _, entry := range page {
			shown := entry.WithheldCopy(visible)
			if csvWriter != nil {
				err = csvWriter.Write(csvRow(shown))
			} else {
				err = writeJSONLine(out, shown)
			}
			if err != nil {
				return nil, err
			}
			stats.rows++
			if stats.rows > maxExportRows {
				return nil, errortypes.NewBusinessError(
					"The export has more than 2,000,000 entries; narrow the range or filters",
				)
			}
		}
		if csvWriter != nil {
			csvWriter.Flush()
			if err = csvWriter.Error(); err != nil {
				return nil, err
			}
		}
		if heartbeat != nil {
			heartbeat(strconv.FormatInt(stats.rows, 10))
		}
		if len(page) < exportPageSize {
			return stats, nil
		}
		last := page[len(page)-1]
		afterTS, afterID = last.Timestamp, last.ID
	}
}

func csvRow(e *audit.Entry) []string {
	changes := ""
	if len(e.Changes) > 0 {
		if raw, err := sonic.MarshalString(e.Changes); err == nil {
			changes = raw
		}
	}
	return []string{
		e.ID.String(),
		time.Unix(e.Timestamp, 0).UTC().Format(time.RFC3339),
		e.Action,
		e.Resource.String(),
		e.ResourceID,
		string(e.Operation),
		string(e.Category),
		string(e.RetentionClass),
		string(e.Channel),
		e.PrincipalType,
		e.PrincipalID.String(),
		e.ActorName,
		e.UserID.String(),
		e.APIKeyID.String(),
		e.AIRunID,
		e.EntityRef,
		e.Comment,
		e.IPAddress,
		e.UserAgent,
		e.CorrelationID,
		e.TraceID,
		strconv.FormatBool(e.Critical),
		strconv.FormatBool(e.SensitiveData),
		changes,
	}
}

func writeJSONLine(out io.Writer, e *audit.Entry) error {
	raw, err := sonic.Marshal(e)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	_, err = out.Write(raw)
	return err
}

func (s *Service) fail(ctx context.Context, export *audit.Export, message string) (*audit.Export, error) {
	now := s.now().Unix()
	export.Status = audit.ExportStatusFailed
	export.ErrorMessage = truncate(message, 1000)
	export.CompletedAt = &now
	updated, err := s.exports.Update(ctx, export)
	if err != nil {
		return nil, err
	}
	s.metrics.Audit.RecordExport(string(export.Format), false)
	s.notify(ctx, updated)
	return updated, nil
}

type Download struct {
	URL       string
	FileName  string
	ExpiresAt int64
	SHA256    string
}

func (s *Service) Download(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor kernelports.AuditActor,
	exportID pulid.ID,
) (*Download, error) {
	export, err := s.exports.GetByID(ctx, auditports.GetExportRequest{ID: exportID, TenantInfo: tenant})
	if err != nil {
		return nil, err
	}
	if export.RequestedByID != tenant.UserID {
		return nil, errortypes.NewAuthorizationError("Only the person who asked for an export may download it")
	}

	now := s.now()
	if export.Status == audit.ExportStatusExpired ||
		(export.ArtifactExpiresAt != nil && *export.ArtifactExpiresAt <= now.Unix()) {
		return nil, errortypes.NewBusinessError("This export has expired; export the log again")
	}
	if !export.Downloadable(now.Unix()) {
		return nil, errortypes.NewBusinessError("This export has no file to download yet")
	}

	fileName := export.FileName()
	url, err := s.storage.GetPresignedURL(ctx, &kernelports.PresignedURLParams{
		Key:                export.ArtifactKey,
		Expiry:             downloadURLExpiry,
		ContentDisposition: fmt.Sprintf("attachment; filename=%q", fileName),
	})
	if err != nil {
		s.l.Error("failed to presign an audit export", zap.String("exportId", export.ID.String()), zap.Error(err))
		return nil, errortypes.NewBusinessError("The export could not be prepared for download; try again shortly")
	}

	s.record(ctx, actor, export, audit.ActionExportDownloaded, "Audit log export downloaded")
	return &Download{
		URL:       url,
		FileName:  fileName,
		ExpiresAt: now.Add(downloadURLExpiry).Unix(),
		SHA256:    export.SHA256,
	}, nil
}

func (s *Service) ExpireDue(ctx context.Context) (int, error) {
	due, err := s.exports.ListExpired(ctx, s.now().Unix(), cleanupBatchSize)
	if err != nil {
		return 0, err
	}
	expired := 0
	for _, export := range due {
		tenantCtx := ctx
		if delErr := s.storage.DeleteObject(tenantCtx, &kernelports.DeleteObjectParams{
			Key: export.ArtifactKey,
		}); delErr != nil {
			s.l.Warn("failed to delete an expired audit export file",
				zap.String("exportId", export.ID.String()), zap.Error(delErr))
			continue
		}
		export.Status = audit.ExportStatusExpired
		export.ArtifactKey = ""
		if _, err = s.exports.Update(withTenant(ctx, export), export); err != nil {
			return expired, err
		}
		expired++
	}
	return expired, nil
}

func (s *Service) record(
	ctx context.Context,
	actor kernelports.AuditActor,
	export *audit.Export,
	action string,
	comment string,
) {
	if s.auditor == nil {
		return
	}
	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceAuditLog,
		ResourceID:     export.ID.String(),
		Operation:      permission.OpExport,
		Action:         action,
		Actor:          actor,
		OrganizationID: export.OrganizationID,
		BusinessUnitID: export.BusinessUnitID,
		Comment:        comment,
		Metadata: map[string]any{
			"format":          string(export.Format),
			"rangeFrom":       export.RangeFrom,
			"rangeTo":         export.RangeTo,
			"includeSecurity": export.IncludeSecurity,
			"rowCount":        export.RowCount,
		},
	})
}

func (s *Service) notify(ctx context.Context, export *audit.Export) {
	if s.notifications == nil {
		return
	}
	userID := export.RequestedByID
	buID := export.BusinessUnitID
	correlation := export.ID.String()
	title := "Audit log export ready"
	message := fmt.Sprintf("Your export of %d audit entries is ready to download.", export.RowCount)
	priority := notification.PriorityMedium
	if export.Status == audit.ExportStatusFailed {
		title = "Audit log export failed"
		message = "Your audit log export could not be written: " + export.ErrorMessage
		priority = notification.PriorityHigh
	}
	if _, err := s.notifications.Create(ctx, &notification.Notification{
		OrganizationID: export.OrganizationID,
		BusinessUnitID: &buID,
		TargetUserID:   &userID,
		Channel:        notification.ChannelUser,
		EventType:      "audit.export." + string(export.Status),
		Priority:       priority,
		Title:          title,
		Message:        message,
		Source:         noticeSource,
		CorrelationID:  &correlation,
		Data:           map[string]any{"link": "/admin/audit-logs?tab=exports", "exportId": correlation},
	}); err != nil {
		s.l.Warn("failed to notify about an audit export", zap.String("exportId", correlation), zap.Error(err))
	}
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
