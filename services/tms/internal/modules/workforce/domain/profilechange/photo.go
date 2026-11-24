package profilechange

import (
	"slices"

	"github.com/emoss08/trenova/pkg/errortypes"
)

func PhotoChange() []FieldChange {
	return []FieldChange{{Field: ProfileFieldPhoto}}
}

func (r *WorkerProfileChangeRequest) IsPhoto() bool {
	return slices.ContainsFunc(r.Changes, func(change FieldChange) bool {
		return change.Field == ProfileFieldPhoto
	})
}

func (r *WorkerProfileChangeRequest) HasProposedPhoto() bool {
	return r.ProposedPhotoKey != ""
}

func (r *WorkerProfileChangeRequest) ProposePhoto(key, thumbnailKey string) {
	r.ProposedPhotoKey = key
	r.ProposedPhotoThumbnailKey = thumbnailKey
}

func (r *WorkerProfileChangeRequest) validatePhoto(multiErr *errortypes.MultiError) {
	if !r.IsPhoto() {
		if r.HasProposedPhoto() {
			multiErr.Add("changes", errortypes.ErrInvalid, "Only a photo request can carry a photo")
		}
		return
	}
	if len(r.Changes) > 1 {
		multiErr.Add(
			"changes",
			errortypes.ErrInvalid,
			"A photo is asked for on its own, without other changes",
		)
	}
	if r.Status == ProfileChangePending && !r.HasProposedPhoto() {
		multiErr.Add("photo", errortypes.ErrRequired, "A photo request needs the proposed photo")
	}
}
