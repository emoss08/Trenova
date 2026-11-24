package driversettlement

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type HoldReason string

const (
	HoldReasonRateDispute      = HoldReason("RateDispute")
	HoldReasonMissingPaperwork = HoldReason("MissingPaperwork")
	HoldReasonNegativeNet      = HoldReason("NegativeNet")
	HoldReasonDeductionReview  = HoldReason("DeductionReview")
	HoldReasonOther            = HoldReason("Other")
)

const MaxHoldNoteLength = 500

const fieldHoldReason = "holdReason"

const fieldHoldNote = "holdNote"

func HoldReasons() []HoldReason {
	return []HoldReason{
		HoldReasonRateDispute,
		HoldReasonMissingPaperwork,
		HoldReasonNegativeNet,
		HoldReasonDeductionReview,
		HoldReasonOther,
	}
}

func (r HoldReason) String() string { return string(r) }

func (r HoldReason) IsValid() bool {
	return slices.Contains(HoldReasons(), r)
}

func HoldableStatuses() []Status {
	return UnreleasedStatuses()
}

type HoldInput struct {
	Reason HoldReason
	Note   string
	UserID pulid.ID
	At     int64
}

func (s *Settlement) IsHeld() bool { return s.HoldReason != "" }

func CheckHoldInput(reason HoldReason, note string) error {
	if !reason.IsValid() {
		return errortypes.NewValidationError(
			fieldHoldReason,
			errortypes.ErrInvalid,
			"Choose why the settlement is held",
		)
	}
	note = strings.TrimSpace(note)
	if reason == HoldReasonOther && note == "" {
		return errortypes.NewValidationError(
			fieldHoldNote,
			errortypes.ErrRequired,
			"Say why the settlement is held when the reason is Other",
		)
	}
	if utf8.RuneCountInString(note) > MaxHoldNoteLength {
		return errortypes.NewValidationError(
			fieldHoldNote,
			errortypes.ErrInvalid,
			"The hold note must be 500 characters or fewer",
		)
	}

	return nil
}

func (s *Settlement) checkHoldable() error {
	switch s.Status {
	case StatusDraft, StatusPendingApproval, StatusApproved:
		return nil
	case StatusPosted, StatusPaid:
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"A posted settlement is paid through its journal; void it instead of holding it",
		)
	case StatusVoided:
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"A voided settlement cannot be held",
		)
	}

	return errortypes.NewValidationError(
		"status",
		errortypes.ErrInvalidOperation,
		"Settlement status is invalid",
	)
}

func (s *Settlement) PlaceHold(input *HoldInput) error {
	if err := CheckHoldInput(input.Reason, input.Note); err != nil {
		return err
	}
	if err := s.checkHoldable(); err != nil {
		return err
	}
	if s.IsHeld() {
		return errortypes.NewValidationError(
			fieldHoldReason,
			errortypes.ErrInvalidOperation,
			"Settlement {0} is already on hold; release it before holding it again",
			s.SettlementNumber,
		)
	}

	at := input.At
	s.HoldReason = input.Reason
	s.HoldNote = strings.TrimSpace(input.Note)
	s.HeldByID = input.UserID
	s.HeldAt = &at
	s.HoldReleasedByID = pulid.Nil
	s.HoldReleasedAt = nil

	return nil
}

func (s *Settlement) ReleaseHold(userID pulid.ID, at int64) error {
	if !s.IsHeld() {
		return errortypes.NewValidationError(
			fieldHoldReason,
			errortypes.ErrInvalidOperation,
			"Settlement {0} is not on hold",
			s.SettlementNumber,
		)
	}
	s.clearHold()
	s.HoldReleasedByID = userID
	s.HoldReleasedAt = &at

	return nil
}

func (s *Settlement) DropHold() {
	s.clearHold()
}

func (s *Settlement) clearHold() {
	s.HoldReason = ""
	s.HoldNote = ""
	s.HeldByID = pulid.Nil
	s.HeldAt = nil
}

func (s *Settlement) RefuseIfHeld() error {
	if !s.IsHeld() {
		return nil
	}

	return heldError(s.HoldReason, s.SettlementNumber)
}

func heldError(reason HoldReason, number string) error {
	switch reason {
	case HoldReasonRateDispute:
		return errortypes.NewValidationError(fieldHoldReason, errortypes.ErrInvalidOperation,
			"Settlement {0} is on hold for a rate dispute; release the hold first", number)
	case HoldReasonMissingPaperwork:
		return errortypes.NewValidationError(fieldHoldReason, errortypes.ErrInvalidOperation,
			"Settlement {0} is on hold for missing paperwork; release the hold first", number)
	case HoldReasonNegativeNet:
		return errortypes.NewValidationError(fieldHoldReason, errortypes.ErrInvalidOperation,
			"Settlement {0} is on hold for a negative net; release the hold first", number)
	case HoldReasonDeductionReview:
		return errortypes.NewValidationError(fieldHoldReason, errortypes.ErrInvalidOperation,
			"Settlement {0} is on hold while a deduction is reviewed; release the hold first",
			number)
	case HoldReasonOther:
	}

	return errortypes.NewValidationError(fieldHoldReason, errortypes.ErrInvalidOperation,
		"Settlement {0} is on hold; release the hold first", number)
}

func (s *Settlement) validateHold(multiErr *errortypes.MultiError) {
	if !s.IsHeld() {
		if s.HeldAt != nil || s.HoldNote != "" {
			multiErr.Add(
				fieldHoldReason,
				errortypes.ErrInvalid,
				"Choose why the settlement is held",
			)
		}

		return
	}
	if !s.HoldReason.IsValid() {
		multiErr.Add(fieldHoldReason, errortypes.ErrInvalid, "Choose why the settlement is held")
	}
	if s.HeldAt == nil {
		multiErr.Add("heldAt", errortypes.ErrRequired, "A held settlement records when it was held")
	}
	if utf8.RuneCountInString(s.HoldNote) > MaxHoldNoteLength {
		multiErr.Add(
			fieldHoldNote,
			errortypes.ErrInvalid,
			"The hold note must be 500 characters or fewer",
		)
	}
	if !slices.Contains(HoldableStatuses(), s.Status) {
		multiErr.Add(
			"status",
			errortypes.ErrInvalidOperation,
			"A posted settlement is paid through its journal; void it instead of holding it",
		)
	}
}
