package reportjobs

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	needsAttentionNotification = "report_needs_attention"
	needsAttentionSource       = "reportjobs.CheckDefinitionsAgainstCatalog"
	needsAttentionCorrelation  = "report-catalog-check:"
	maxNoticeDefinitionIDs     = 20
	dataKeyReportCount         = "reportCount"
	dataKeyDefinitionIDs       = "definitionIds"
)

type attentionOwner struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	OwnerID        pulid.ID
}

type attentionNotice struct {
	Owner         attentionOwner
	FirstReport   string
	Count         int
	DefinitionIDs []pulid.ID
}

type attentionNotices struct {
	byOwner map[attentionOwner]*attentionNotice
}

func newAttentionNotices() *attentionNotices {
	return &attentionNotices{byOwner: make(map[attentionOwner]*attentionNotice)}
}

func (n *attentionNotices) add(def *report.ReportDefinition) {
	owner := attentionOwner{
		OrganizationID: def.OrganizationID,
		BusinessUnitID: def.BusinessUnitID,
		OwnerID:        def.OwnerID,
	}
	notice, ok := n.byOwner[owner]
	if !ok {
		notice = &attentionNotice{Owner: owner, FirstReport: def.Name}
		n.byOwner[owner] = notice
	}
	notice.Count++
	if len(notice.DefinitionIDs) < maxNoticeDefinitionIDs {
		notice.DefinitionIDs = append(notice.DefinitionIDs, def.ID)
	}
}

func (n *attentionNotices) list() []*attentionNotice {
	out := make([]*attentionNotice, 0, len(n.byOwner))
	for _, notice := range n.byOwner {
		out = append(out, notice)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].Owner, out[j].Owner
		if a.OrganizationID != b.OrganizationID {
			return a.OrganizationID < b.OrganizationID
		}
		if a.BusinessUnitID != b.BusinessUnitID {
			return a.BusinessUnitID < b.BusinessUnitID
		}
		return a.OwnerID < b.OwnerID
	})
	return out
}

func (a *Activities) notifyNeedsAttention(
	ctx context.Context,
	version string,
	notices *attentionNotices,
) int {
	sent := 0
	for _, notice := range notices.list() {
		if a.notifyOwnerNeedsAttention(ctx, version, notice) {
			sent++
		}
	}
	return sent
}

func (a *Activities) notifyOwnerNeedsAttention(
	ctx context.Context,
	version string,
	notice *attentionNotice,
) bool {
	tenantInfo := pagination.TenantInfo{
		OrgID:  notice.Owner.OrganizationID,
		BuID:   notice.Owner.BusinessUnitID,
		UserID: notice.Owner.OwnerID,
	}
	tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())

	wording, ok := a.renderNotification(
		tenantCtx,
		tenantInfo,
		documenttemplate.KindNotificationReportNeedsAttention,
		notice.DefinitionIDs[0],
		&documenttemplate.ReportNotificationContext{
			ReportName:   notice.FirstReport,
			ReportCount:  notice.Count,
			OtherReports: notice.Count - 1,
		},
	)
	if !ok {
		return false
	}

	ids := make([]string, 0, len(notice.DefinitionIDs))
	for _, id := range notice.DefinitionIDs {
		ids = append(ids, id.String())
	}
	correlation := needsAttentionCorrelation + version + ":" + notice.Owner.OwnerID.String()
	buID := notice.Owner.BusinessUnitID
	ownerID := notice.Owner.OwnerID

	if _, err := a.notification.Create(tenantCtx, &notification.Notification{
		OrganizationID: notice.Owner.OrganizationID,
		BusinessUnitID: &buID,
		TargetUserID:   &ownerID,
		Channel:        notification.ChannelUser,
		EventType:      needsAttentionNotification,
		Priority:       notification.PriorityHigh,
		Title:          wording.Title,
		Message:        wording.Message,
		Source:         needsAttentionSource,
		CorrelationID:  &correlation,
		Data: map[string]any{
			dataKeyReportName:    notice.FirstReport,
			dataKeyReportCount:   notice.Count,
			dataKeyDefinitionIDs: ids,
		},
	}); err != nil {
		a.l.Warn("failed to tell a report owner their reports need attention",
			zap.String("ownerId", ownerID.String()), zap.Error(err))
		return false
	}

	return true
}
