package boardhint

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

var shows = map[boardcatalog.Form]string{
	boardcatalog.FormTrend:   "how it moved over time",
	boardcatalog.FormBars:    "how they rank or compare",
	boardcatalog.FormColumns: "each one against its allowance",
	boardcatalog.FormSplit:   "how the whole divides",
	boardcatalog.FormGrid:    "where it is covered and where it falls short",
	boardcatalog.FormDots:    "how they spread",
	boardcatalog.FormSpans:   "how long each one lasts",
	boardcatalog.FormLog:     "what happened, in order",
	boardcatalog.FormChecks:  "what stands in the way",
	boardcatalog.FormTable:   "every figure side by side",
	boardcatalog.FormFigures: "these figures together",
}

func Shows(form boardcatalog.Form) string {
	return shows[form]
}

type Records struct {
	Words   string
	Measure string
	From    string
}

type Show struct {
	Metric  string
	For     pulid.ID
	Period  string
	Records *Records
}

func Line(show *Show) string {
	if show.Metric == "" && show.Records != nil {
		return recordsLine(show.Records)
	}
	metric, known := boardregistry.Catalog().MetricNamed(show.Metric)
	if !known {
		return ""
	}
	show = withRecords(show, &metric)
	groups := []string{metricGroup(&metric, show)}
	if show.Records != nil {
		groups = append(groups, recordsGroup())
	}
	var b strings.Builder
	fmt.Fprintf(&b, Lead+" %s rather than one figure, do not "+
		"answer in prose or a list: answer with show_answer %s.", Shows(metric.Form),
		call(firstFigure(&metric), measureOf(show.Records), groups))
	if show.Records != nil {
		fmt.Fprintf(&b, " The second group holds %s", show.Records.Words)
		if show.Records.From != "" {
			fmt.Fprintf(&b, ", read with %s", show.Records.From)
		}
		b.WriteString(", by their ids." + ListAsked)
	}
	if show.Period == "" && len(metric.Periods) > 1 {
		fmt.Fprintf(&b, " Its period is the one the question asks about: %s.",
			periodList(&metric))
	}
	b.WriteString(figureList(&metric))
	fmt.Fprintf(&b, " Place only the figures the question asks about, at most %d in all; "+
		"the board draws the rest.", board.MaxReadings)
	if follow := FollowUpOf(measureOf(show.Records)); follow != nil {
		b.WriteString(" " + FollowUpLine(follow))
	}
	if show.Records == nil {
		b.WriteString(" {count} does not apply: the group holds no records.")
	}

	return b.String()
}

func withRecords(show *Show, metric *boardcatalog.Metric) *Show {
	if show.Records != nil || metric.Records == nil {
		return show
	}
	filled := *show
	filled.Records = &Records{
		Words:   metric.Records.Words,
		Measure: metric.Records.Measure,
		From:    metric.Records.From,
	}

	return &filled
}

func recordsLine(records *Records) string {
	var b strings.Builder
	fmt.Fprintf(&b, Lead+" %s weighed together rather than one of "+
		"them, do not answer in prose or a list: answer with show_answer %s, the group "+
		"holding %s by their ids.", records.Words,
		call("", records.Measure, []string{recordsGroup()}), records.Words)
	b.WriteString(ListAsked)
	if records.Measure != "" {
		fmt.Fprintf(&b, " Each is drawn with its %s; write {total:%s} or {largest:%s} where "+
			"a figure across them goes, and {%s:NUMBER} for one of them, never typed.",
			records.Measure, records.Measure, records.Measure, records.Measure)
	}
	if follow := FollowUpOf(records.Measure); follow != nil {
		b.WriteString(" " + FollowUpLine(follow))
	}

	return b.String()
}

func Brief(show *Show) string {
	metric, known := boardregistry.Catalog().MetricNamed(show.Metric)
	if !known {
		return ""
	}

	return fmt.Sprintf("To show %s, answer with show_answer and a group %s.",
		Shows(metric.Form), metricGroup(&metric, show))
}

func metricGroup(metric *boardcatalog.Metric, show *Show) string {
	group := fmt.Sprintf(`{"title": "…", "metric": %q`, metric.Name)
	switch {
	case show.For.IsNotNil():
		group += fmt.Sprintf(`, "for": %q`, show.For.String())
	case metric.For != "":
		group += fmt.Sprintf(`, "for": "the %s's id"`, metric.For.Noun())
	}
	switch {
	case show.Period != "" && show.Period != string(metric.DefaultPeriod()):
		group += fmt.Sprintf(`, "period": %q`, show.Period)
	case show.Period == "" && len(metric.Periods) > 1:
		group += `, "period": "…"`
	}

	return group + "}"
}

func recordsGroup() string {
	return `{"title": "…", "records": ["…"]}`
}

func measureOf(records *Records) string {
	if records == nil {
		return ""
	}

	return records.Measure
}

func call(figure, measure string, groups []string) string {
	measured := ""
	if measure != "" {
		measured = fmt.Sprintf(`"measure": %q, `, measure)
	}
	offered := ""
	if follow := FollowUpOf(measure); follow != nil {
		offered = fmt.Sprintf(`, "actions": [{"label": %q, "ask": %q}]`, follow.Label, follow.Ask)
	}

	return fmt.Sprintf(`{"title": "…", "answer": "…%s …", %s"groups": [%s]%s}`, figure, measured,
		strings.Join(groups, ", "), offered)
}

func FollowUpOf(measure string) *boardcatalog.FollowUp {
	if measure == "" {
		return nil
	}
	measures := boardregistry.Catalog().Measures()
	for idx := range measures {
		if measures[idx].Name == measure && measures[idx].FollowUp != nil {
			return measures[idx].FollowUp
		}
	}

	return nil
}

func FollowUpLine(follow *boardcatalog.FollowUp) string {
	return fmt.Sprintf("Offer what the person will send as the board's action, "+
		"{\"actions\": [{\"label\": %q, \"ask\": %q}]}, and write no message in your reply: "+
		"the draft is made when they pick it.", follow.Label, follow.Ask)
}

const Lead = "When the person needs to see"

const ListAsked = " When the person asked to see the list to go through it themselves, point " +
	"to its table and draw nothing."

func firstFigure(metric *boardcatalog.Metric) string {
	leads, _ := metric.Led()
	if len(leads) == 0 {
		return ""
	}

	return " {" + leads[0].Name + ":" + metric.Name + "}"
}

func figureList(metric *boardcatalog.Metric) string {
	leads, rest := metric.Led()
	if len(leads) == 0 {
		return ""
	}
	figures := make([]string, 0, len(leads))
	for _, figure := range leads {
		label := figure.Label
		if label != "" {
			label = strings.ToLower(label[:1]) + label[1:]
		}
		figures = append(figures, fmt.Sprintf("{%s:%s} (%s)", figure.Name, metric.Name, label))
	}
	if len(rest) == 0 {
		return fmt.Sprintf(" Its figures, written into the answer instead of typed numbers: %s.",
			strings.Join(figures, ", "))
	}
	others := make([]string, 0, len(rest))
	for _, figure := range rest {
		others = append(others, figure.Name)
	}

	return fmt.Sprintf(" The figures an answer leads with, written into it instead of typed "+
		"numbers: %s. It also keeps %s, which the board draws; place one only when the "+
		"question asks for it.", strings.Join(figures, ", "), strings.Join(others, ", "))
}

func periodList(metric *boardcatalog.Metric) string {
	periods := make([]string, 0, len(metric.Periods))
	for idx, period := range metric.Periods {
		if idx == 0 {
			periods = append(periods, string(period)+" (the default)")
			continue
		}
		periods = append(periods, string(period))
	}

	return strings.Join(periods, ", ")
}

func SubjectLine(id pulid.ID) (line, metric string) {
	read, known := subjectMetric(id)
	if !known {
		return "", ""
	}

	return Line(&Show{Metric: read.Name, For: id}), read.Name
}

func subjectMetric(id pulid.ID) (boardcatalog.Metric, bool) {
	kind, known := recordkind.OfIDPrefix(id.Prefix())
	if !known {
		return boardcatalog.Metric{}, false
	}
	metrics := boardregistry.Catalog().Metrics()
	for idx := range metrics {
		if metrics[idx].For == kind {
			return metrics[idx], true
		}
	}

	return boardcatalog.Metric{}, false
}

func Subject(id pulid.ID) string {
	kind, known := recordkind.OfIDPrefix(id.Prefix())
	if !known {
		return ""
	}
	metrics := boardregistry.Catalog().Metrics()
	for idx := range metrics {
		metric := &metrics[idx]
		if metric.For != kind {
			continue
		}
		hint := fmt.Sprintf("; it is the %s the %s metric reads, so give it as that metric "+
			"group's for: {\"metric\": %q, \"for\": %q}", kind.Noun(), metric.Name, metric.Name,
			id.String())
		if metric.Records != nil {
			hint += fmt.Sprintf(", and list %s in a records group, by the ids %s returns",
				metric.Records.Words, metric.Records.From)
		}

		return hint
	}

	return ""
}
