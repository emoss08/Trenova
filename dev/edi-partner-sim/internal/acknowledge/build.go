package acknowledge

import (
	"errors"
	"slices"
	"strconv"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

const (
	Accepted           = "A"
	AcceptedWithErrors = "E"
	Rejected           = "R"
	Partial            = "P"
)

type Input struct {
	GroupID      string
	GroupControl string
	Version      string
	Transactions []InputTransaction
}

type InputTransaction struct {
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

func Code(rules *partner.Acknowledgment, set string, errors, warnings int) string {
	if slices.Contains(rules.Reject.Sets, set) {
		return Rejected
	}
	switch rules.Reject.When {
	case partner.RejectAny:
		if errors+warnings > 0 {
			return Rejected
		}
	case partner.RejectErrors:
		if errors > 0 {
			return Rejected
		}
		if warnings > 0 {
			return AcceptedWithErrors
		}
	default:
		if errors+warnings > 0 {
			return AcceptedWithErrors
		}
	}
	return Accepted
}

func Build(rules *partner.Acknowledgment, input Input) (partner.Transaction, error) {
	if rules.Sends == partner.AckNone {
		return partner.Transaction{}, errors.New("the partner sends no acknowledgment")
	}
	five := rules.Sends == partner.Ack999
	header := single(input.GroupID, input.GroupControl)
	if five {
		header = single(input.GroupID, input.GroupControl, input.Version)
	}
	segments := []partner.Segment{{ID: "AK1", Elements: header}}
	accepted := 0
	for _, transaction := range input.Transactions {
		segments = append(
			segments,
			partner.Segment{ID: "AK2", Elements: single(transaction.Set, transaction.Control)},
		)
		segments = append(segments, notes(five, transaction.Notes)...)
		segments = append(
			segments,
			partner.Segment{ID: pick(five, "IK5", "AK5"), Elements: single(transaction.Code)},
		)
		if transaction.Code != Rejected {
			accepted++
		}
	}
	segments = append(segments, partner.Segment{ID: "AK9", Elements: single(
		groupCode(input.Transactions, accepted),
		strconv.Itoa(len(input.Transactions)),
		strconv.Itoa(len(input.Transactions)),
		strconv.Itoa(accepted),
	)})
	return partner.Transaction{Set: string(rules.Sends), Segments: segments}, nil
}

func notes(five bool, segmentNotes []SegmentNote) []partner.Segment {
	var segments []partner.Segment
	for _, note := range segmentNotes {
		segments = append(segments, partner.Segment{ID: pick(five, "IK3", "AK3"), Elements: single(
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
			segments = append(segments, partner.Segment{
				ID:       pick(five, "IK4", "AK4"),
				Elements: [][]string{position, {element.Reference}, {element.Code}, {element.Copy}},
			})
		}
	}
	return segments
}

func groupCode(transactions []InputTransaction, accepted int) string {
	switch {
	case accepted == 0 && len(transactions) > 0:
		return Rejected
	case accepted < len(transactions):
		return Partial
	}
	for _, transaction := range transactions {
		if transaction.Code == AcceptedWithErrors {
			return AcceptedWithErrors
		}
	}
	return Accepted
}

func pick(condition bool, yes, no string) string {
	if condition {
		return yes
	}
	return no
}

func single(values ...string) [][]string {
	out := make([][]string, 0, len(values))
	for _, value := range values {
		out = append(out, []string{value})
	}
	return out
}
