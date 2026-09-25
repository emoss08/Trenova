package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxAccountingSyncActionIDs  = 500
	maxAccountingSyncStateIDs   = 200
	accountingSyncIDsField      = "ids"
	accountingSyncObjectIDField = "objectIds"
)

func accountingSyncRecordIDs(raw []string) ([]pulid.ID, error) {
	if len(raw) > maxAccountingSyncActionIDs {
		return nil, errortypes.NewValidationError(
			accountingSyncIDsField,
			errortypes.ErrInvalid,
			"At most {0} records can be changed at a time",
			maxAccountingSyncActionIDs,
		)
	}
	return parseIDs(raw)
}

func accountingSyncObjectIDs(raw []string) ([]pulid.ID, error) {
	if len(raw) > maxAccountingSyncStateIDs {
		return nil, errortypes.NewValidationError(
			accountingSyncObjectIDField,
			errortypes.ErrInvalid,
			"At most {0} documents can be looked up at a time",
			maxAccountingSyncStateIDs,
		)
	}
	return parseIDs(raw)
}

func orderedAccountingSyncStates(
	ids []pulid.ID,
	states map[pulid.ID]*services.AccountingSyncObjectState,
) []*services.AccountingSyncObjectState {
	ordered := make([]*services.AccountingSyncObjectState, 0, len(states))
	seen := make(map[pulid.ID]struct{}, len(states))
	for _, id := range ids {
		state, ok := states[id]
		if !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ordered = append(ordered, state)
	}
	return ordered
}

func accountingSyncRecordConnectionToModel(
	result *pagination.CursorListResult[*accountingsync.AccountingSyncRecord],
) (*gqlmodel.AccountingSyncRecordConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *accountingsync.AccountingSyncRecord, cursor string) *gqlmodel.AccountingSyncRecordEdge {
			return &gqlmodel.AccountingSyncRecordEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AccountingSyncRecordEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AccountingSyncRecordConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
