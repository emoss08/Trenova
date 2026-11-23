package desktranscriptexport

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/storage/uploadpipe"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/objectkey"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	ExportTTL         = 7 * 24 * time.Hour
	conversationsRead = 25
	staleExportAge    = 6 * time.Hour
	cleanupBatchSize  = 100
	errorLimit        = 1000
	noticeSource      = "desktranscriptexport.Exports"
	contentType       = "application/zip"
)

type Heartbeat func(details ...any)

func beat(heartbeat Heartbeat, details ...any) {
	if heartbeat != nil {
		heartbeat(details...)
	}
}

type Notifier interface {
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
}

type Exports struct {
	exports       desktranscriptports.ExportRepository
	conversations desktranscriptports.ConversationRepository
	assembler     *Assembler
	organizations tenantports.OrganizationReader
	storage       kernelports.Client
	notifier      Notifier
	realtime      realtimeports.RealtimeService
	now           func() time.Time
	l             *zap.Logger
}

type ExportsParams struct {
	Exports       desktranscriptports.ExportRepository
	Conversations desktranscriptports.ConversationRepository
	Assembler     *Assembler
	Organizations tenantports.OrganizationReader
	Storage       kernelports.Client
	Notifier      Notifier
	Realtime      realtimeports.RealtimeService
	Now           func() time.Time
	Logger        *zap.Logger
}

func NewExports(p *ExportsParams) *Exports {
	now := p.Now
	if now == nil {
		now = time.Now
	}

	return &Exports{
		exports:       p.Exports,
		conversations: p.Conversations,
		assembler:     p.Assembler,
		organizations: p.Organizations,
		storage:       p.Storage,
		notifier:      p.Notifier,
		realtime:      p.Realtime,
		now:           now,
		l:             p.Logger.Named("desk-transcripts.exports"),
	}
}

func (x *Exports) Run(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	exportID pulid.ID,
	heartbeat Heartbeat,
) (*desktranscript.Export, error) {
	export, err := x.exports.GetByID(ctx, desktranscriptports.GetExportRequest{
		TenantInfo: tenantInfo,
		ID:         exportID,
	})
	if err != nil {
		return nil, err
	}
	if export.Status.Terminal() {
		return export, nil
	}

	if export.Status != desktranscript.ExportStatusRunning || export.StartedAt == nil {
		started := x.now().Unix()
		export.Status = desktranscript.ExportStatusRunning
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
		x.l.Error("failed to write a desk transcript export",
			zap.String("exportId", export.ID.String()), zap.Error(runErr))

		return x.Fail(ctx, export, failureMessage(ctx, runErr))
	}

	now := x.now().Unix()
	expires := now + int64(ExportTTL.Seconds())
	export.Status = desktranscript.ExportStatusSucceeded
	export.ConversationCount = stats.Conversations
	export.ByteSize = stats.Bytes
	export.SHA256 = stats.SHA256
	export.ArtifactKey = key
	export.ArtifactExpiresAt = &expires
	export.CompletedAt = &now
	export.ErrorMessage = ""

	updated, err := x.exports.Update(ctx, export)
	if err != nil {
		return nil, err
	}
	x.notify(ctx, updated)
	x.invalidate(ctx, updated)

	return updated, nil
}

func failureMessage(ctx context.Context, err error) string {
	if errors.Is(err, uploadpipe.ErrTooLarge) {
		return i18n.T(
			ctx,
			"The file passed the {0} MB an export may hold. Export a shorter timeframe, one person or one agent.",
			desktranscript.MaxExportBytes>>20,
		)
	}

	return i18n.T(
		ctx,
		"The export could not be written. Try again; if it fails again, export a shorter timeframe.",
	)
}

func (x *Exports) write(
	ctx context.Context,
	export *desktranscript.Export,
	heartbeat Heartbeat,
) (*ArchiveStats, string, error) {
	tenantInfo := pagination.TenantInfo{OrgID: export.OrganizationID, BuID: export.BusinessUnitID}
	loc, err := tenantports.OrganizationLocation(ctx, x.organizations, tenantInfo)
	if err != nil {
		return nil, "", err
	}

	key := objectkey.DeskTranscriptExports.Key(export.OrganizationID, export.ID.String()+".zip")
	pipe := uploadpipe.Open(ctx, x.storage, uploadpipe.Params{
		Key:         key,
		ContentType: contentType,
		MaxBytes:    desktranscript.MaxExportBytes,
		Metadata: map[string]string{
			"organization-id": export.OrganizationID.String(),
			"export-id":       export.ID.String(),
		},
	})

	stats, err := x.fill(ctx, pipe, export, &RenderBatch{
		TenantInfo: tenantInfo,
		Location:   loc,
		ExportedAt: x.now().Unix(),
	}, heartbeat)
	if err != nil {
		pipe.Abort(err)

		return nil, "", err
	}
	if _, err = pipe.Close(); err != nil {
		return nil, "", fmt.Errorf("upload a desk transcript export: %w", err)
	}

	return stats, key, nil
}

func (x *Exports) fill(
	ctx context.Context,
	pipe *uploadpipe.Pipe,
	export *desktranscript.Export,
	batch *RenderBatch,
	heartbeat Heartbeat,
) (*ArchiveStats, error) {
	zipped, err := openArchive(pipe.Writer(), x.now())
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := zipped.close(); closeErr != nil {
			x.l.Warn("failed to remove a desk transcript export's working file",
				zap.String("exportId", export.ID.String()), zap.Error(closeErr))
		}
	}()

	for chunk := range slices.Chunk(export.ThreadIDs, conversationsRead) {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		threads, listErr := x.conversations.ListByIDs(
			ctx,
			desktranscriptports.ConversationsByIDsRequest{TenantInfo: batch.TenantInfo, IDs: chunk},
		)
		if listErr != nil {
			return nil, listErr
		}
		batch.Threads = inOrder(chunk, threads)
		if err = x.assembler.Render(ctx, batch, func(
			_ *conversation.Thread,
			rendered *desktranscriptports.RenderedConversation,
		) error {
			return zipped.add(rendered)
		}); err != nil {
			return nil, err
		}
		beat(heartbeat, zipped.written)
	}

	return zipped.finish()
}

func inOrder(ids []pulid.ID, threads []*conversation.Thread) []*conversation.Thread {
	byID := make(map[pulid.ID]*conversation.Thread, len(threads))
	for _, thread := range threads {
		byID[thread.ID] = thread
	}
	ordered := make([]*conversation.Thread, 0, len(threads))
	for _, id := range ids {
		if thread, ok := byID[id]; ok {
			ordered = append(ordered, thread)
		}
	}

	return ordered
}

func (x *Exports) Fail(
	ctx context.Context,
	export *desktranscript.Export,
	message string,
) (*desktranscript.Export, error) {
	now := x.now().Unix()
	export.Status = desktranscript.ExportStatusFailed
	export.ErrorMessage = stringutils.TruncateBytes(message, errorLimit)
	export.CompletedAt = &now

	updated, err := x.exports.Update(ctx, export)
	if err != nil {
		return nil, err
	}
	x.notify(ctx, updated)
	x.invalidate(ctx, updated)

	return updated, nil
}

func (x *Exports) notify(ctx context.Context, export *desktranscript.Export) {
	if x.notifier == nil {
		return
	}

	eventType := desktranscriptports.ExportReadyEvent
	priority := notification.PriorityMedium
	title := i18n.T(ctx, "Your export of Desk conversations is ready")
	message := i18n.T(
		ctx,
		"Open Desk conversations in administration to download it. It can be downloaded for seven days.",
	)
	if export.Status == desktranscript.ExportStatusFailed {
		eventType = desktranscriptports.ExportFailedEvent
		priority = notification.PriorityHigh
		title = i18n.T(ctx, "Your export of Desk conversations failed")
		message = export.ErrorMessage
	}

	buID := export.BusinessUnitID
	userID := export.RequestedByUserID
	correlation := export.ID.String() + ":" + string(export.Status)

	if _, err := x.notifier.Create(ctx, &notification.Notification{
		OrganizationID: export.OrganizationID,
		BusinessUnitID: &buID,
		TargetUserID:   &userID,
		Channel:        notification.ChannelUser,
		EventType:      eventType,
		Priority:       priority,
		Title:          title,
		Message:        message,
		Source:         noticeSource,
		CorrelationID:  &correlation,
		Data: map[string]any{
			"kind":              eventType,
			"exportId":          export.ID.String(),
			"status":            string(export.Status),
			"conversationCount": export.ConversationCount,
			"link":              desktranscriptports.ExportsPath,
		},
		RelatedEntities: map[string]any{"deskTranscriptExportId": export.ID.String()},
	}); err != nil {
		x.l.Warn("failed to tell a person their desk transcript export finished",
			zap.String("exportId", export.ID.String()), zap.Error(err))
	}
}

func (x *Exports) invalidate(ctx context.Context, export *desktranscript.Export) {
	if x.realtime == nil {
		return
	}

	if err := x.realtime.PublishResourceInvalidation(ctx,
		&realtimeports.PublishResourceInvalidationRequest{
			OrganizationID: export.OrganizationID,
			BusinessUnitID: export.BusinessUnitID,
			Resource:       desktranscriptports.ExportResource,
			Action:         "updated",
			RecordID:       export.ID,
			EntityVersion:  export.Version,
			ActorUserID:    export.RequestedByUserID,
		}); err != nil {
		x.l.Warn("failed to announce a desk transcript export",
			zap.String("exportId", export.ID.String()), zap.Error(err))
	}
}

func tenantContext(ctx context.Context, export *desktranscript.Export) context.Context {
	return dbscope.WithValidTenant(ctx, pagination.TenantInfo{
		OrgID: export.OrganizationID,
		BuID:  export.BusinessUnitID,
	}.DBTenant())
}

type CleanupResult struct {
	Expired int `json:"expired"`
	Failed  int `json:"failed"`
}

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
			exportCtx := tenantContext(ctx, export)
			if err = x.storage.Delete(exportCtx, export.ArtifactKey); err != nil {
				x.l.Warn("failed to delete an expired desk transcript export",
					zap.String("exportId", export.ID.String()), zap.Error(err))

				continue
			}
			export.Status = desktranscript.ExportStatusExpired
			export.ArtifactKey = ""
			if _, err = x.exports.Update(exportCtx, export); err != nil {
				x.l.Warn("failed to mark a desk transcript export expired",
					zap.String("exportId", export.ID.String()), zap.Error(err))

				continue
			}
			progressed = true
			result.Expired++
			x.invalidate(exportCtx, export)
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
		if _, err = x.Fail(
			tenantContext(ctx, export),
			export,
			i18n.T(ctx, "The export was abandoned before it finished. Export again."),
		); err != nil {
			x.l.Warn("failed to fail an abandoned desk transcript export",
				zap.String("exportId", export.ID.String()), zap.Error(err))

			continue
		}
		result.Failed++
	}

	return result, nil
}
