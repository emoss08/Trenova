package toolpreview

import (
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

// ChainStep is one step of a plan and its preview, computed against the
// world as it is before any step runs.
type ChainStep struct {
	Step    int
	Preview *agent.ProposalPreview
}

type chainKey struct {
	resource permission.Resource
	id       pulid.ID
}

type projectedValue struct {
	step  int
	value any
	ref   *agent.PreviewRef
}

type chainedRecord struct {
	step   int
	fields map[string]projectedValue
}

// Chain projects a plan: a step that changes a record an earlier step also
// changes depends on that step, and a value the earlier step sets is what
// the later step starts from, not what the record holds now. Steps are read
// in the order given. Records the plan creates have no id yet and are never
// chained.
func Chain(steps []ChainStep) {
	touched := make(map[chainKey]*chainedRecord, len(steps))

	for _, step := range steps {
		preview := step.Preview
		if preview == nil {
			continue
		}

		for i := range preview.Changes {
			change := &preview.Changes[i]
			if change.EntityID.IsNil() || change.Operation == agent.PreviewOperationCreate {
				continue
			}
			key := chainKey{resource: change.Resource, id: change.EntityID}

			if earlier, ok := touched[key]; ok {
				change.DependsOnStep = earlier.step
				for j := range change.Fields {
					field := &change.Fields[j]
					if value, set := earlier.fields[field.Path]; set {
						field.Before = value.value
						field.BeforeRef = cloneRef(value.ref)
						field.ProjectedFromStep = value.step
					}
				}
				preview.AddWarning(agent.PreviewWarning{
					Code: agent.PreviewWarningDependsOnStep,
					Args: []string{strconv.Itoa(earlier.step)},
					Message: "This step changes a record step " + strconv.Itoa(earlier.step) +
						" changes first; it is shown as it would be after that step.",
				})
			}

			record, ok := touched[key]
			if !ok {
				record = &chainedRecord{fields: make(map[string]projectedValue, len(change.Fields))}
				touched[key] = record
			}
			record.step = step.Step
			for j := range change.Fields {
				field := &change.Fields[j]
				record.fields[field.Path] = projectedValue{
					step:  step.Step,
					value: field.After,
					ref:   cloneRef(field.AfterRef),
				}
			}
		}
	}
}

func cloneRef(ref *agent.PreviewRef) *agent.PreviewRef {
	if ref == nil {
		return nil
	}
	clone := *ref
	if ref.Record != nil {
		record := *ref.Record
		clone.Record = &record
	}

	return &clone
}
