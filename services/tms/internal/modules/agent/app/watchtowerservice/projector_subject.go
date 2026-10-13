package watchtowerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/zap"
)

const subjectItemLimit = 200

func AsSubjectReconciler(p *Projector) watchtowerports.WatchtowerSubjectReconciler { return p }

func (p *Projector) ReconcileSubject(
	ctx context.Context,
	req *watchtowerports.ReconcileSubjectRequest,
) {
	if req == nil || req.SubjectID.IsNil() {
		return
	}

	open := make(map[string]struct{}, len(req.Open))
	for i := range req.Open {
		input := req.Open[i]
		if input.SourceKind != req.SourceKind || input.SubjectID != req.SubjectID {
			continue
		}
		p.Upsert(ctx, input)
		open[input.SourceID] = struct{}{}
	}

	scoped := dbscope.WithTenant(ctx, req.TenantInfo.DBTenant())
	existing, err := p.repo.List(scoped, watchtowerports.ListWatchtowerItemsQuery{
		TenantInfo:     req.TenantInfo,
		Kinds:          []watchtower.SourceKind{req.SourceKind},
		UnresolvedOnly: true,
		SubjectID:      req.SubjectID,
		Limit:          subjectItemLimit,
	})
	if err != nil {
		p.l.Warn("could not read a record's watchtower items",
			zap.String("kind", string(req.SourceKind)),
			zap.String("subject", req.SubjectID.String()),
			zap.Error(err))
		return
	}
	for _, item := range existing {
		if _, ok := open[item.SourceID]; !ok {
			p.Resolve(ctx, req.TenantInfo, item.SourceKind, item.SourceID)
		}
	}
}
