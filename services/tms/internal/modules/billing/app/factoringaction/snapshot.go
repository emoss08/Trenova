package factoringaction

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/shared/pulid"
)

func UniqueIDs(ids []pulid.ID) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(ids))
	out := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		if id.IsNil() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func CloneSubmission(src *factoringsubmission.Submission) *factoringsubmission.Submission {
	copied := *src
	copied.Items = make([]*factoringsubmission.SubmissionItem, 0, len(src.Items))
	for _, item := range src.Items {
		if item == nil {
			continue
		}
		itemCopy := *item
		copied.Items = append(copied.Items, &itemCopy)
	}
	return &copied
}
