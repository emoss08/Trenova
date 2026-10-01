package resolver

import (
	"sort"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/shared/pulid"
)

func employmentValues(values map[string]string) []*gqlmodel.WorkerEmploymentValue {
	out := make([]*gqlmodel.WorkerEmploymentValue, 0, len(values))
	for key, value := range values {
		out = append(out, &gqlmodel.WorkerEmploymentValue{Key: key, Value: value})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// optionalIDPtr distinguishes "not provided" (nil) from "cleared" (empty
// string), which a transfer uses to unassign a fleet.
func optionalIDPtr(value *string) (*pulid.ID, error) {
	if value == nil {
		return nil, nil //nolint:nilnil // absence is meaningful here
	}
	if *value == "" {
		id := pulid.Nil
		return &id, nil
	}
	id, err := pulid.MustParse(*value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
