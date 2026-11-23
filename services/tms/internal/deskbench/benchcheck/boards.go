package benchcheck

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchresult"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
)

type shownBoard struct {
	id      string
	title   string
	payload *board.Payload
}

func turnBoards(turn *benchrecord.TurnRecord) []shownBoard {
	boards := make([]shownBoard, 0, 1)
	for idx := range turn.Artifacts {
		artifact := &turn.Artifacts[idx]
		if artifact.Kind != assistantartifact.KindBoard {
			continue
		}
		payload, err := board.PayloadOf(artifact.Payload)
		if err != nil {
			continue
		}
		boards = append(boards, shownBoard{
			id:      artifact.ID.String(),
			title:   artifact.Title,
			payload: payload,
		})
	}

	return boards
}

func noBoardCheck(turns []*benchrecord.TurnRecord) benchresult.CheckResult {
	shown := make([]string, 0, 1)
	for _, turn := range turns {
		for idx := range turn.Artifacts {
			if turn.Artifacts[idx].Kind == assistantartifact.KindBoard {
				shown = append(shown, strconv.Quote(turn.Artifacts[idx].Title))
			}
		}
	}

	return benchresult.CheckResult{
		Name:   "no board was shown",
		Passed: len(shown) == 0,
		Detail: strings.Join(shown, "; "),
	}
}

func boardCheck(
	want *benchscenario.ExpectedBoard,
	turns []*benchrecord.TurnRecord,
) benchresult.CheckResult {
	place := board.Place(strings.TrimSpace(want.Place))
	name := "answered with a board"
	if place.IsValid() {
		name += " placed " + string(place)
	}
	if want.MinReadings > 0 {
		name += fmt.Sprintf(" whose words place at least %d readings", want.MinReadings)
	}
	if len(want.Reads) > 0 {
		name += fmt.Sprintf(", reading %s each from a reading of its own measure",
			strings.Join(want.Reads, " and "))
	}
	check := benchresult.CheckResult{Name: name}

	seen := make([]string, 0, 2)
	for _, turn := range turns {
		for _, shown := range turnBoards(turn) {
			readings := len(shown.payload.Board.Readings)
			seen = append(seen, fmt.Sprintf("%q %s with %d readings%s",
				shown.title, shown.payload.Place, readings, readMeasures(&shown.payload.Board)))
			if place.IsValid() && shown.payload.Place != place {
				continue
			}
			if readings < want.MinReadings || !readsEach(&shown.payload.Board, want.Reads) {
				continue
			}
			if shown.payload.Place == board.PlaceConversation &&
				!slices.Contains(replycheck.ArtifactRefIDs(turn.Reply), shown.id) {
				continue
			}
			check.Passed = true
			return check
		}
	}
	if len(seen) == 0 {
		check.Detail = "no board was shown"
	} else {
		check.Detail = "shown: " + strings.Join(seen, "; ")
	}

	return check
}

func readsEach(layout *board.Board, measures []string) bool {
	return !slices.ContainsFunc(measures, func(measure string) bool {
		return !slices.ContainsFunc(layout.Readings, func(reading board.Reading) bool {
			return readsValue(&reading.FigureSource) && string(reading.Measure) == measure
		})
	})
}

func readsValue(source *board.FigureSource) bool {
	return source.Measure != "" &&
		(source.Aggregate == board.AggregateSum || source.Aggregate == board.AggregateMax)
}

func readMeasures(layout *board.Board) string {
	read := make([]string, 0, len(layout.Readings))
	for idx := range layout.Readings {
		source := &layout.Readings[idx].FigureSource
		if readsValue(source) && !slices.Contains(read, string(source.Measure)) {
			read = append(read, string(source.Measure))
		}
	}
	if len(read) == 0 {
		return ""
	}

	return " of " + strings.Join(read, " and ")
}
