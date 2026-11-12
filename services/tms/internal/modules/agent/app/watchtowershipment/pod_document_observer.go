package watchtowershipment

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentexceptionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const shipmentDocumentResource = "shipment"

// PodDocumentObserver settles a shipment's missing POD the moment its papers
// change. A paper filed on a shipment is not a shipment event, so without
// this the item, and the case it opened, stayed in front of the person who
// had just uploaded the proof of delivery until the next sweep. A POD that
// leaves the shipment raises the item again the same way.
//
// An agent handed the missing paper often raises an exception of its own
// saying the same thing: it cannot produce a document, so it asks a person
// to. That exception is about the papers too, so once the shipment has its
// POD and its billing asks for no other document, it is settled with the
// item. One that is about something else (a rate, a reference) is left.
type PodDocumentObserver struct {
	exceptions shipmentExceptions
	reconciler watchtowerports.WatchtowerSubjectReconciler
	raised     agentexceptionports.AgentExceptionRepository
	settle     agentexceptionports.AgentExceptionService
	readiness  shipmentserviceports.ShipmentBillingReadinessReader
	l          *zap.Logger
}

type PodDocumentObserverParams struct {
	fx.In

	ShipmentExceptionParams

	Reconciler watchtowerports.WatchtowerSubjectReconciler
	Raised     agentexceptionports.AgentExceptionRepository
	Settle     agentexceptionports.AgentExceptionService
	Readiness  shipmentserviceports.ShipmentBillingReadinessReader
	Logger     *zap.Logger
}

const maxPaperExceptions = 25

const papersOnFileNote = "The documents this shipment's billing asks for are now on file."

var _ documentports.DocumentChangeObserver = (*PodDocumentObserver)(nil)

func NewPodDocumentObserver(
	p PodDocumentObserverParams,
) documentports.DocumentChangeObserver {
	return &PodDocumentObserver{
		exceptions: newShipmentExceptions(p.ShipmentExceptionParams),
		reconciler: p.Reconciler,
		raised:     p.Raised,
		settle:     p.Settle,
		readiness:  p.Readiness,
		l:          p.Logger.Named("watchtower.pod-documents"),
	}
}

func (o *PodDocumentObserver) DocumentsChanged(
	ctx context.Context,
	change *documentports.DocumentChange,
) {
	if change == nil || change.ResourceType != shipmentDocumentResource {
		return
	}
	shipmentIDs := make([]pulid.ID, 0, len(change.ResourceIDs))
	for _, raw := range change.ResourceIDs {
		if id, err := pulid.Parse(raw); err == nil && id.IsNotNil() {
			shipmentIDs = append(shipmentIDs, id)
		}
	}
	if len(shipmentIDs) == 0 {
		return
	}

	ctx = dbscope.WithTenant(ctx, change.TenantInfo.DBTenant())
	open, err := o.exceptions.forKind(
		ctx,
		change.TenantInfo,
		watchtower.SourcePodMissing,
		shipmentIDs,
	)
	if err != nil {
		o.l.Warn("could not settle missing PODs after a shipment's papers changed", zap.Error(err))

		return
	}

	bySubject := make(map[pulid.ID][]watchtowerports.WatchtowerItemInput, len(open))
	for idx := range open {
		bySubject[open[idx].SubjectID] = append(bySubject[open[idx].SubjectID], open[idx])
	}
	for _, shipmentID := range shipmentIDs {
		o.reconciler.ReconcileSubject(ctx, &watchtowerports.ReconcileSubjectRequest{
			TenantInfo: change.TenantInfo,
			SourceKind: watchtower.SourcePodMissing,
			SubjectID:  shipmentID,
			Open:       bySubject[shipmentID],
		})
		if len(bySubject[shipmentID]) == 0 {
			o.settlePaperExceptions(ctx, change, shipmentID)
		}
	}
}

func (o *PodDocumentObserver) settlePaperExceptions(
	ctx context.Context,
	change *documentports.DocumentChange,
	shipmentID pulid.ID,
) {
	readiness, err := o.readiness.GetBillingReadiness(ctx, shipmentID, change.TenantInfo)
	if err != nil {
		o.l.Warn("could not read which documents a shipment still needs", zap.Error(err))

		return
	}
	if len(readiness.MissingRequirements) > 0 {
		return
	}

	found, err := o.raised.List(ctx, &agentexceptionports.ListAgentExceptionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: change.TenantInfo,
			Pagination: pagination.Info{Limit: maxPaperExceptions},
			FieldFilters: []domaintypes.FieldFilter{
				{Field: "subjectId", Operator: dbtype.OpEqual, Value: shipmentID.String()},
				{
					Field:    "resolutionState",
					Operator: dbtype.OpEqual,
					Value:    string(agentexception.ResolutionStateOpen),
				},
			},
		},
	})
	if err != nil {
		o.l.Warn("could not read the exceptions raised about a shipment's papers", zap.Error(err))

		return
	}
	for _, raised := range found.Items {
		if raised == nil || !raised.Category.AboutDocuments() {
			continue
		}
		if _, err = o.settle.Resolve(ctx, &agentexceptionports.ResolveAgentExceptionRequest{
			ID:              raised.ID,
			ResolutionState: agentexception.ResolutionStateResolved,
			ResolutionNotes: papersOnFileNote,
			TenantInfo:      change.TenantInfo,
		}, nil); err != nil {
			o.l.Warn("could not settle an exception about a shipment's papers",
				zap.String("exception", raised.ID.String()), zap.Error(err))
		}
	}
}
