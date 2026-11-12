package agenttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/caseowner"
	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// OpenCases is case ownership with every case unowned, for a test whose
// subject only needs to ask. Owned names the records someone else owns, which
// it refuses to work and does not show.
type OpenCases struct {
	Owned  map[pulid.ID]pulid.ID
	Routed []caseownerports.CandidatesRequest
}

var _ caseownerports.Service = (*OpenCases)(nil)

func (c *OpenCases) Sight(
	_ context.Context,
	actor *permissionports.RequestActor,
) (caseowner.Sight, error) {
	return caseowner.Sight{As: []pulid.ID{actor.UserID}}, nil
}

func (c *OpenCases) Standing(
	_ context.Context,
	req caseownerports.RecordRequest,
	actor *permissionports.RequestActor,
) (*caseownerports.Standing, error) {
	owner, owned := c.Owned[req.RecordID]
	if !owned {
		return &caseownerports.Standing{CanWork: true, CanSee: true, CanClaim: true}, nil
	}
	mine := owner == actor.UserID

	return &caseownerports.Standing{
		Owner:   &caseowner.Owner{RecordID: req.RecordID, OwnerID: owner},
		CanWork: mine,
		CanSee:  mine,
	}, nil
}

func (c *OpenCases) Standings(
	ctx context.Context,
	req caseownerports.RecordsRequest,
	actor *permissionports.RequestActor,
) (map[pulid.ID]*caseownerports.Standing, error) {
	out := make(map[pulid.ID]*caseownerports.Standing, len(req.RecordIDs))
	for _, id := range req.RecordIDs {
		standing, err := c.Standing(ctx, caseownerports.RecordRequest{RecordID: id}, actor)
		if err != nil {
			return nil, err
		}
		out[id] = standing
	}

	return out, nil
}

func (c *OpenCases) AssertMayWork(
	ctx context.Context,
	req caseownerports.RecordRequest,
	actor *permissionports.RequestActor,
) error {
	standing, err := c.Standing(ctx, req, actor)
	if err != nil {
		return err
	}
	if !standing.CanWork {
		return errortypes.NewAuthorizationError("This case is owned by someone else")
	}

	return nil
}

func (c *OpenCases) Claim(
	_ context.Context,
	req *caseownerports.AssignRequest,
	actor *permissionports.RequestActor,
) (*caseowner.Owner, error) {
	return &caseowner.Owner{RecordID: req.RecordID, OwnerID: actor.UserID}, nil
}

func (c *OpenCases) Assign(
	_ context.Context,
	req *caseownerports.AssignRequest,
	_ *permissionports.RequestActor,
) (*caseowner.Owner, error) {
	return &caseowner.Owner{RecordID: req.RecordID, OwnerID: req.OwnerID}, nil
}

func (c *OpenCases) Release(
	context.Context,
	*caseownerports.ReleaseRequest,
	*permissionports.RequestActor,
) error {
	return nil
}

func (c *OpenCases) Route(
	_ context.Context,
	req caseownerports.CandidatesRequest,
	_ ...pulid.ID,
) {
	c.Routed = append(c.Routed, req)
}
