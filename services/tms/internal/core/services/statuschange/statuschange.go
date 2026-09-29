package statuschange

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type Request[T any, S ~string] struct {
	Field  string
	Kind   string
	IDs    []pulid.ID
	Found  []*T
	Status S
	Valid  bool
	IDOf   func(*T) pulid.ID
	Set    func(*T, S)
}

func Plan[T any, S ~string](req *Request[T, S]) ([]services.RecordChange[T], error) {
	multiErr := errortypes.NewMultiError()
	if len(req.IDs) == 0 {
		multiErr.Add(req.Field, errortypes.ErrRequired, "Name at least one {0}", req.Kind)
	}
	if !req.Valid {
		multiErr.Add("status", errortypes.ErrInvalid, "{0} is not a {1} status",
			string(req.Status), req.Kind)
	}

	found := make(map[pulid.ID]*T, len(req.Found))
	for _, entity := range req.Found {
		found[req.IDOf(entity)] = entity
	}

	missing := make([]string, 0)
	for _, id := range req.IDs {
		if _, ok := found[id]; !ok {
			missing = append(missing, id.String())
		}
	}
	if len(missing) > 0 {
		multiErr.Add(req.Field, errortypes.ErrInvalid,
			"These are not {0} records of this organization: {1}",
			req.Kind, strings.Join(missing, ", "))
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	changes := make([]services.RecordChange[T], 0, len(req.Found))
	for _, entity := range req.Found {
		after := *entity
		req.Set(&after, req.Status)
		changes = append(changes, services.RecordChange[T]{Before: entity, After: &after})
	}

	return changes, nil
}
