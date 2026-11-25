package runtimetools

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/boardrecords"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardanswer"
	"github.com/emoss08/trenova/shared/pulid"
)

type counted struct {
	answer    *boardanswer.Answer
	arguments map[string]any
	repaired  bool
	refusal   string
}

func countedWhole(
	read boardrecords.Read,
	answer *boardanswer.Answer,
	arguments map[string]any,
) counted {
	out := counted{answer: answer, arguments: arguments}
	if whole := answer.Board.Reading(boardanswer.NameCount); whole != nil && whole.Records != nil {
		ids := whole.Records.IDs
		total, longer := read.Longer(ids)
		if longer && !read.WholeTyped(answer.Words, len(ids)) {
			if !boardrecords.PointsAtShown(answer.Words, "{count}") {
				out.refusal = boardrecords.CountRefusal(ShowAnswerTool, "{count}",
					whole.Records.Kind, len(ids), total)
				return out
			}
		}
	}
	if out.refusal = groupCounts(read, out.answer); out.refusal != "" {
		return out
	}
	listed, table, refused := wholeList(read, out.answer)
	if out.refusal = refused; out.refusal != "" {
		return out
	}
	if listed && out.retranslate(boardrecords.AllOf(out.arguments, table)) {
		return out
	}
	out.refusal = sumOfAPage(read, &out.answer.Board)

	return out
}

func (c *counted) retranslate(arguments map[string]any) bool {
	answer, problems := boardanswer.Translate(arguments)
	c.arguments, c.repaired = arguments, true
	if len(problems) > 0 {
		c.refusal = fmt.Sprintf("Tool %q was not run: %s.", ShowAnswerTool,
			strings.Join(problems, "; "))
		return true
	}
	c.answer = answer

	return false
}

func groupCounts(read boardrecords.Read, answer *boardanswer.Answer) string {
	for idx := range answer.Board.Readings {
		reading := &answer.Board.Readings[idx]
		if !strings.HasPrefix(reading.Key, boardanswer.NameCount+"_") || reading.Records == nil {
			continue
		}
		ids := reading.Records.IDs
		total, longer := read.Longer(ids)
		if !longer || read.WholeTyped(answer.Words, len(ids)) {
			continue
		}
		mark := "{" + reading.Key + "}"
		if boardrecords.PointsAtShown(answer.Words, mark) {
			continue
		}
		return boardrecords.CountRefusal(ShowAnswerTool, mark, reading.Records.Kind, len(ids),
			total)
	}

	return ""
}

func sumOfAPage(read boardrecords.Read, layout *board.Board) string {
	texts := []string{layout.Verdict}
	for idx := range layout.Blocks {
		texts = append(texts, layout.Blocks[idx].Caption)
	}
	words := strings.Join(texts, "\n")
	for idx := range layout.Readings {
		reading := &layout.Readings[idx]
		if reading.Records == nil || (reading.Aggregate != board.AggregateSum &&
			reading.Aggregate != board.AggregateMax) {
			continue
		}
		total, tied := read.SumTiedToWhole(words, "{"+reading.Key+"}", reading.Records.IDs)
		if !tied {
			continue
		}
		return fmt.Sprintf("Tool %q was not run: {%s} reads only the %d records shown, but "+
			"its words tie it to the whole list (%d in all), so it would state a figure that is "+
			"not true. Say it is the shown records' (\"these {count} bring in {%s}\"), or "+
			"leave it out.", ShowAnswerTool, reading.Key, len(reading.Records.IDs), total,
			reading.Key)
	}

	return ""
}

func wholeList(
	read boardrecords.Read,
	answer *boardanswer.Answer,
) (listed bool, table pulid.ID, refusal string) {
	all := answer.Board.Reading(boardanswer.NameAll)
	if all == nil || all.Records == nil {
		return false, pulid.Nil, ""
	}
	ids := all.Records.IDs
	named, elsewhere, refusal := read.AllNamed(ShowAnswerTool, answer.Words, all.Records.Kind,
		ids)
	if refusal != "" || elsewhere {
		return elsewhere, named.Table, refusal
	}
	whole, kept := read.WholeOf(ids)
	if !kept {
		if _, longer := read.Longer(ids); longer {
			return false, pulid.Nil, boardrecords.UnkeptRefusal(ShowAnswerTool)
		}
		return false, pulid.Nil, ""
	}

	return true, whole.Table, ""
}
