package palettequestion

import (
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// IDPrefix names a palette question; its workflow and stream are keyed
	// on the same id.
	IDPrefix = "pask_"
	// MaxAsking is how many questions one person may have Nova answering at
	// once. The palette shows one answer, so more are only cost.
	MaxAsking = 2
	// AskingWindow is how long a question still marked as being answered
	// counts against MaxAsking; a worker lost mid-answer frees its slot after
	// it.
	AskingWindow = 5 * 60
	// KeepClaimWindow is how long one request carrying a question into the
	// Desk holds it before another may try.
	KeepClaimWindow = 60
)

type Status string

const (
	StatusAsking   = Status("Asking")
	StatusAnswered = Status("Answered")
	StatusFailed   = Status("Failed")
	StatusStopped  = Status("Stopped")
)

func AllStatuses() []Status {
	return []Status{StatusAsking, StatusAnswered, StatusFailed, StatusStopped}
}

func (s Status) IsValid() bool { return slices.Contains(AllStatuses(), s) }

// Question is a question asked from the command palette. It holds nothing of
// what was asked or answered; it is what the person's AI allowance counts,
// and it records the Desk conversation the question was carried into.
type Question struct {
	keel.Table `keel:"assistant_palette_questions"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:pask_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// UserID is the person who asked.
	UserID pulid.ID `json:"userId" keel:"user_id"`
	// Status is how far the answer got.
	Status Status `json:"status" keel:"status"`
	// KeptThreadID is the Desk conversation the question was carried into.
	KeptThreadID pulid.ID `json:"keptThreadId" keel:"kept_thread_id,nullzero"`
	// KeepClaimedAt is when carrying it into the Desk began.
	KeepClaimedAt *int64 `json:"keepClaimedAt" keel:"keep_claimed_at"`
	// FinishedAt is when the answer ended, however it ended.
	FinishedAt *int64 `json:"finishedAt" keel:"finished_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
}
