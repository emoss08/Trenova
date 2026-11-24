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
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
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
	if len(want.Kinds) > 0 {
		name += ", showing records of " + strings.Join(want.Kinds, " and ")
	}
	if len(want.Measures) > 0 {
		name += ", its rows measured by " + strings.Join(want.Measures, " and ")
	}
	if len(want.Metrics) > 0 {
		name += ", reading the metrics " + strings.Join(want.Metrics, " and ")
	}
	if len(want.Figures) > 0 {
		name += ", placing " + strings.Join(want.Figures, " and ")
	}
	if want.Links {
		name += ", its words linking an artifact the turn made"
	}
	if want.Actions {
		name += ", offering an action or linking the draft the turn made"
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
			layout := &shown.payload.Board
			if readings < want.MinReadings || !readsEach(layout, want.Reads) ||
				!showsKinds(layout, want.Kinds) || !measuresRows(layout, want.Measures) ||
				!readsMetrics(layout, want.Metrics) ||
				!placesFigures(layout, want.Figures) || !linksMade(layout, turn, want.Links) ||
				(want.Actions && !offersAction(layout) && !linksDraft(layout, turn)) {
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

func showsKinds(layout *board.Board, kinds []string) bool {
	return !slices.ContainsFunc(kinds, func(kind string) bool {
		return !slices.ContainsFunc(layout.Blocks, func(block board.Block) bool {
			return block.Type == board.BlockRecords && !block.Records.Empty() &&
				string(block.Records.Kind) == kind
		})
	})
}

func measuresRows(layout *board.Board, measures []string) bool {
	return !slices.ContainsFunc(measures, func(measure string) bool {
		return !slices.ContainsFunc(layout.Blocks, func(block board.Block) bool {
			return block.Type == board.BlockRecords && string(block.Measure) == measure
		})
	})
}

func readsMetrics(layout *board.Board, metrics []string) bool {
	asked := layout.Metrics()

	return !slices.ContainsFunc(metrics, func(metric string) bool {
		return !slices.ContainsFunc(asked, func(ask board.MetricAsk) bool {
			return ask.Metric == metric
		})
	})
}

func placesFigures(layout *board.Board, figures []string) bool {
	return !slices.ContainsFunc(figures, func(figure string) bool {
		name, metric, _ := strings.Cut(figure, ":")
		return !slices.ContainsFunc(layout.Readings, func(reading board.Reading) bool {
			return reading.Metric == metric && strings.EqualFold(reading.Figure, name)
		})
	})
}

func linksMade(layout *board.Board, turn *benchrecord.TurnRecord, wanted bool) bool {
	links := layout.ArtifactLinks()
	if wanted && len(links) == 0 {
		return false
	}

	return !slices.ContainsFunc(links, func(id string) bool {
		return !slices.ContainsFunc(
			turn.Artifacts,
			func(made assistantartifactports.AssistantArtifact) bool {
				return made.ID.String() == id && made.Kind != assistantartifact.KindBoard
			},
		)
	})
}

func linksDraft(layout *board.Board, turn *benchrecord.TurnRecord) bool {
	return slices.ContainsFunc(layout.ArtifactLinks(), func(id string) bool {
		return slices.ContainsFunc(
			turn.Artifacts,
			func(made assistantartifactports.AssistantArtifact) bool {
				return made.ID.String() == id && made.Kind == assistantartifact.KindDocument
			},
		)
	})
}

func offersAction(layout *board.Board) bool {
	return slices.ContainsFunc(layout.Blocks, func(block board.Block) bool {
		return block.Type == board.BlockActions && len(block.Actions) > 0
	})
}
