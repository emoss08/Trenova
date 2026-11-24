package replycheck

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardhint"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

const (
	minSetOutItems        = 2
	subjectCorrectionLead = "Your reply sets out in prose what a metric Trenova keeps draws:"
)

var setOutItem = regexp.MustCompile(`^\s*(?:[-*•+]|\d{1,2}[.)])\s+\S|` +
	`^\s*(?:\*\*|__)[^*_\n]{1,80}(?:\*\*|__)`)

type subjectHint struct {
	metric string
	hint   string
}

func resultHint(content string) string {
	own := toolresult.HowToShow(content)
	id, about := toolresult.Subject(content)
	if !about || strings.Contains(own, id.String()) {
		return own
	}
	line, metric := boardhint.SubjectLine(id)
	if line == "" || (own != "" && !strings.Contains(own, strconv.Quote(metric))) {
		return own
	}

	return line
}

func turnSubject(t Turn) subjectHint {
	messages := t.Messages()
	for idx := len(messages) - 1; idx >= 0; idx-- {
		message := &messages[idx]
		if message.Role == llmports.RoleUser {
			return subjectHint{}
		}
		if message.Role != llmports.RoleTool || message.IsError {
			continue
		}
		id, about := toolresult.Subject(message.Content)
		if !about {
			continue
		}
		line, metric := boardhint.SubjectLine(id)
		if line == "" || !drawsMoreThanFigures(metric) {
			continue
		}
		hint := resultHint(message.Content)
		if !strings.Contains(hint, id.String()) {
			hint = line
		}

		return subjectHint{metric: metric, hint: hint}
	}

	return subjectHint{}
}

func setOutItems(reply string) int {
	count := 0
	for line := range strings.SplitSeq(reply, "\n") {
		if setOutItem.MatchString(line) {
			count++
		}
	}

	return count
}

func weighsInProse(t Turn, reply string) bool {
	if setOutItems(reply) >= minSetOutItems {
		return true
	}
	for paragraph := range strings.SplitSeq(reply, "\n\n") {
		if len(namedBoardRecords(t, paragraph)) >= minBoardRecords {
			return true
		}
	}

	return false
}

func setsOutHinted(t Turn, reply string) bool {
	return t.Request() != nil && strings.Contains(t.Request().Input, "?") &&
		turnHint(t) != "" && setOutItems(reply) >= minSetOutItems
}

func subjectCorrection(subject *subjectHint) string {
	return subjectCorrectionLead + " it sets out several things about the record the " +
		subject.metric + " metric reads, each in its own line. An answer that weighs them " +
		"together belongs on a board in the conversation, not in prose and not as a pointer " +
		"to a table or a card. As this turn's result said: " + subject.hint + " It is drawn " +
		"as your reply and ends your turn, so make any other call first. If the person only " +
		"asked to see a list to go through, or asked for one fact, reply again as you did." +
		UnseenCorrection
}
