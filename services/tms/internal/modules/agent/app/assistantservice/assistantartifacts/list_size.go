package assistantartifacts

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/typeutils"
)

const payloadRecordEntity = "recordEntity"

var errNoTotal = errortypes.NewBusinessError("The list this table came from reports no total")

func (s *Service) ListSize(
	ctx context.Context,
	req *assistantports.ListSizeRequest,
) (*assistantports.ListSize, error) {
	thread := &conversationports.GetThreadRequest{
		ID:         req.ThreadID,
		TenantInfo: req.Actor.TenantInfo(),
	}
	artifact, err := s.artifacts.GetByID(ctx, assistantartifactports.GetArtifactRequest{
		ID:         req.ArtifactID,
		TenantInfo: thread.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if artifact.ThreadID != req.ThreadID || artifact.Kind != assistantartifact.KindTableView ||
		!artifactpayload.Rereadable(artifact) {
		return nil, errNotPageable
	}
	arguments, recorded := artifact.Payload[artifactpayload.PayloadReadWith].(map[string]any)
	if !recorded {
		return nil, errNotReread
	}
	source := &exportSource{artifact: artifact, arguments: arguments}
	list, err := s.listOf(ctx, thread, req.Actor, source)
	if err != nil {
		return nil, err
	}
	if req.Agent != nil {
		list.params.DataAccessCeiling = req.Agent.DataAccess
	}
	page, err := list.read(ctx, map[string]any{argLimit: 1}, true)
	if err != nil {
		return nil, err
	}
	total, counted := typeutils.IntOf(page["total"])
	if !counted {
		return nil, errNoTotal
	}

	return &assistantports.ListSize{
		Title: source.artifact.Title,
		Kind:  typeutils.StringOfTrimmed(source.artifact.Payload[payloadRecordEntity]),
		Total: total,
	}, nil
}
