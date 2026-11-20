package partner

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
)

const (
	AckAccepted           = "A"
	AckAcceptedWithErrors = "E"
	AckRejected           = "R"
	AckPartial            = "P"
)

type AckInput struct {
	GroupID      string
	GroupControl string
	Version      string
	Transactions []AckTransaction
}

type AckTransaction struct {
	Set     string
	Control string
	Code    string
	Notes   []SegmentNote
}

type SegmentNote struct {
	ID       string
	Position int
	Loop     string
	Code     string
	Elements []ElementNote
}

type ElementNote struct {
	Position  int
	Component int
	Reference string
	Code      string
	Copy      string
}

func (a *Acknowledgment) Decide(set string, errors, warnings int) string {
	if slices.Contains(a.Reject.Sets, set) {
		return AckRejected
	}
	switch a.Reject.When {
	case RejectAny:
		if errors+warnings > 0 {
			return AckRejected
		}
	case RejectErrors:
		if errors > 0 {
			return AckRejected
		}
		if warnings > 0 {
			return AckAcceptedWithErrors
		}
	default:
		if errors+warnings > 0 {
			return AckAcceptedWithErrors
		}
	}
	return AckAccepted
}

func (a *Acknowledgment) Silent(rng *rand.Rand) bool {
	return a.Sends == AckNone || a.Never > 0 && rng.Float64() < a.Never
}

func (a *Acknowledgment) Build(input AckInput) (Transaction, error) {
	if a.Sends == AckNone {
		return Transaction{}, fmt.Errorf("the partner sends no acknowledgment")
	}
	five := a.Sends == Ack999
	header := single(input.GroupID, input.GroupControl)
	if five {
		header = single(input.GroupID, input.GroupControl, input.Version)
	}
	segments := []Segment{{ID: "AK1", Elements: header}}
	accepted := 0
	for _, transaction := range input.Transactions {
		segments = append(
			segments,
			Segment{ID: "AK2", Elements: single(transaction.Set, transaction.Control)},
		)
		for _, note := range transaction.Notes {
			segments = append(segments, Segment{ID: pick(five, "IK3", "AK3"), Elements: single(
				note.ID,
				strconv.Itoa(note.Position),
				note.Loop,
				note.Code,
			)})
			for _, element := range note.Elements {
				position := []string{strconv.Itoa(element.Position)}
				if element.Component > 0 {
					position = append(position, strconv.Itoa(element.Component))
				}
				segments = append(
					segments,
					Segment{ID: pick(five, "IK4", "AK4"), Elements: [][]string{
						position,
						{element.Reference},
						{element.Code},
						{element.Copy},
					}},
				)
			}
		}
		segments = append(
			segments,
			Segment{ID: pick(five, "IK5", "AK5"), Elements: single(transaction.Code)},
		)
		if transaction.Code != AckRejected {
			accepted++
		}
	}
	segments = append(segments, Segment{ID: "AK9", Elements: single(
		groupCode(input.Transactions, accepted),
		strconv.Itoa(len(input.Transactions)),
		strconv.Itoa(len(input.Transactions)),
		strconv.Itoa(accepted),
	)})
	return Transaction{Set: string(a.Sends), Segments: segments}, nil
}

func groupCode(transactions []AckTransaction, accepted int) string {
	switch {
	case accepted == 0 && len(transactions) > 0:
		return AckRejected
	case accepted < len(transactions):
		return AckPartial
	}
	for _, transaction := range transactions {
		if transaction.Code == AckAcceptedWithErrors {
			return AckAcceptedWithErrors
		}
	}
	return AckAccepted
}

func pick(condition bool, yes, no string) string {
	if condition {
		return yes
	}
	return no
}
