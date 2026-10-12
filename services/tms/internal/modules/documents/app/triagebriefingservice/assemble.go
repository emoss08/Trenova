package triagebriefingservice

import (
	"math"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	MaxRiverPoints     = 200
	MaxWaitingScanned  = 1000
	MaxDeckItems       = 5
	MaxCarryOver       = 50
	secondsPerHour     = 3600
	secondsPerMinute   = 60
	percent            = 100
	accuracyPrecision  = 10
	handledSampleLimit = 3
)

type Inputs struct {
	Now         int64
	DayStart    int64
	Location    *time.Location
	Settings    *triage.Settings
	Counts      *triageports.BriefingCounts
	Mailboxes   []*inboundmessage.Mailbox
	NovaEnabled bool
	Handled     []*triageports.HandledRow
	Arrivals    []*triageports.ArrivalRow
	River       []*triageports.RiverRow
	Waiting     []*inboundmessage.InboundMessage
	Replies     map[pulid.ID]*triage.Reply
	Corrections *triageports.CorrectionTotals
	Suggestion  *inboundmessageports.RuleSuggestion
}

func Assemble(in *Inputs) *inboundmessageports.Briefing {
	active, unhealthy := sources(in.Mailboxes, in.Now)
	phase := inboundmessageports.BriefingDay
	if timeutils.MinuteOfDay(in.Now, in.Location) >= in.Settings.EndOfDayMinute {
		phase = inboundmessageports.BriefingEndOfDay
	}

	briefing := &inboundmessageports.Briefing{
		GeneratedAt:         in.Now,
		DayStart:            in.DayStart,
		Phase:               phase,
		FirstDay:            active == 0 || !in.Counts.AnyMessage,
		NovaEnabled:         in.NovaEnabled,
		ActiveSources:       active,
		Waiting:             in.Counts.Waiting,
		Overdue:             in.Counts.Overdue,
		ClosedByPeopleToday: in.Counts.ClosedByPeopleToday,
		HandledByNovaToday:  in.Counts.HandledByNovaToday,
		HandledByNova:       HandledGroups(in.Handled),
		Arrivals:            Arrivals(in.Arrivals, in.DayStart, in.Now),
		Corrections:         Corrections(in.Corrections),
		ComingBack:          in.Counts.ComingBack,
		UnhealthySourceIDs:  unhealthy,
		Suggestion:          in.Suggestion,
		OnCallUserID:        in.Settings.OnCallUserID,
		CarryOver:           []*inboundmessageports.DeckItem{},
	}
	briefing.NovaPoints, briefing.Yours = River(in.River)

	deck := Deck(in.Waiting, in.Replies)
	briefing.Deck = deck.cards
	briefing.LongestWaiting = deck.longest
	briefing.EstimateMinutes = deck.estimate
	if phase == inboundmessageports.BriefingEndOfDay {
		briefing.CarryOver = deck.carryOver
	}

	return briefing
}

func sources(mailboxes []*inboundmessage.Mailbox, now int64) (int, []pulid.ID) {
	active := 0
	unhealthy := make([]pulid.ID, 0, 1)
	for _, mailbox := range mailboxes {
		if mailbox.Status == inboundmessage.MailboxActive {
			active++
		}
		if mailbox.Health(now).Unhealthy() {
			unhealthy = append(unhealthy, mailbox.ID)
		}
	}

	return active, unhealthy
}

func HandledGroups(rows []*triageports.HandledRow) []*inboundmessageports.HandledGroup {
	groups := make([]*inboundmessageports.HandledGroup, 0, 8)
	index := make(map[string]*inboundmessageports.HandledGroup, 8)
	for _, row := range rows {
		key := string(row.HandledAction) + "|" + string(row.Classification)
		group, ok := index[key]
		if !ok {
			group = &inboundmessageports.HandledGroup{
				Action:  row.HandledAction,
				Count:   row.GroupCount,
				Samples: make([]*inboundmessageports.Sample, 0, handledSampleLimit),
			}
			if row.Classification != "" {
				class := row.Classification
				group.Classification = &class
			}
			index[key] = group
			groups = append(groups, group)
		}
		if len(group.Samples) < handledSampleLimit {
			group.Samples = append(group.Samples, triagesupport.SampleOf(&row.MessageRow))
		}
	}

	return groups
}

func Arrivals(
	rows []*triageports.ArrivalRow,
	dayStart, now int64,
) []*inboundmessageports.ArrivalBucket {
	hours := 0
	if now > dayStart {
		hours = int((now-dayStart)/secondsPerHour) + 1
	}

	buckets := make([]*inboundmessageports.ArrivalBucket, hours)
	for hour := range buckets {
		buckets[hour] = &inboundmessageports.ArrivalBucket{
			HourStart: dayStart + int64(hour)*secondsPerHour,
		}
	}
	for _, row := range rows {
		if row.Hour < 0 || row.Hour >= hours {
			continue
		}
		buckets[row.Hour].HandledByNova += row.HandledByNova
		buckets[row.Hour].LeftForPerson += row.LeftForPerson
	}

	return buckets
}

func River(
	rows []*triageports.RiverRow,
) (nova, yours []*inboundmessageports.RiverPoint) {
	nova = make([]*inboundmessageports.RiverPoint, 0, len(rows))
	yours = make([]*inboundmessageports.RiverPoint, 0, len(rows))
	for _, row := range rows {
		point := &inboundmessageports.RiverPoint{
			MessageID:   row.ID,
			FromName:    row.FromName,
			FromAddress: row.FromAddress,
			Subject:     row.Subject,
			ReceivedAt:  row.ReceivedAt,
			Done:        row.Done(),
		}
		if row.Classification != "" {
			class := row.Classification
			point.Classification = &class
		}
		if row.DueAt > 0 {
			due := row.DueAt
			point.DueAt = &due
		}

		if row.Nova {
			nova = append(nova, point)
		} else {
			yours = append(yours, point)
		}
	}

	return nova, yours
}

type deckResult struct {
	cards     []*inboundmessageports.DeckCard
	longest   *inboundmessageports.DeckItem
	estimate  int
	carryOver []*inboundmessageports.DeckItem
}

func Deck(
	waiting []*inboundmessage.InboundMessage,
	replies map[pulid.ID]*triage.Reply,
) deckResult {
	byGroup := make(map[triage.DeckGroup]*inboundmessageports.DeckCard, len(triage.AllDeckGroups()))
	result := deckResult{
		cards:     make([]*inboundmessageports.DeckCard, 0, len(triage.AllDeckGroups())),
		carryOver: make([]*inboundmessageports.DeckItem, 0, min(len(waiting), MaxCarryOver)),
	}

	for _, message := range waiting {
		actions := triage.Actions(triage.ActionInput{Message: message, Reply: replies[message.ID]})
		group, next, ok := triage.GroupFor(message, actions)
		item := deckItem(message, &next, ok)
		if !ok {
			group = triage.DeckDecide
		}

		card, exists := byGroup[group]
		if !exists {
			card = &inboundmessageports.DeckCard{
				Group:      group,
				Items:      make([]*inboundmessageports.DeckItem, 0, MaxDeckItems),
				MessageIDs: make([]pulid.ID, 0, 8),
			}
			byGroup[group] = card
		}
		card.Count++
		card.MessageIDs = append(card.MessageIDs, message.ID)
		if len(card.Items) < MaxDeckItems {
			card.Items = append(card.Items, item)
		}
		if len(result.carryOver) < MaxCarryOver {
			result.carryOver = append(result.carryOver, item)
		}
		if result.longest == nil || overdueFirst(item, result.longest) {
			result.longest = item
		}
	}

	for _, group := range triage.AllDeckGroups() {
		card, ok := byGroup[group]
		if !ok {
			continue
		}
		card.EstimateMinutes = int(math.Ceil(
			float64(card.Count*group.EstimateSeconds()) / secondsPerMinute,
		))
		result.estimate += card.EstimateMinutes
		result.cards = append(result.cards, card)
	}

	return result
}

func deckItem(
	message *inboundmessage.InboundMessage,
	next *triage.Action,
	hasNext bool,
) *inboundmessageports.DeckItem {
	item := &inboundmessageports.DeckItem{
		MessageID:   message.ID,
		FromName:    message.FromName,
		FromAddress: message.FromAddress,
		Subject:     message.Subject,
		ReceivedAt:  message.ReceivedAt,
		OwnerID:     message.OwnerID,
	}
	if message.Classification != "" {
		class := message.Classification
		item.Classification = &class
	}
	if message.DueAt > 0 {
		due := message.DueAt
		item.DueAt = &due
	}
	if hasNext {
		action := *next
		item.Action = &action
	}

	return item
}

func overdueFirst(candidate, current *inboundmessageports.DeckItem) bool {
	switch {
	case candidate.DueAt != nil && current.DueAt == nil:
		return true
	case candidate.DueAt == nil && current.DueAt != nil:
		return false
	case candidate.DueAt != nil && *candidate.DueAt != *current.DueAt:
		return *candidate.DueAt < *current.DueAt
	default:
		return candidate.ReceivedAt < current.ReceivedAt
	}
}

func Corrections(totals *triageports.CorrectionTotals) inboundmessageports.CorrectionsSummary {
	this := accuracy(totals.ReadThisWeek, totals.CorrectedThisWeek)
	summary := inboundmessageports.CorrectionsSummary{
		ThisWeek: totals.CorrectionsThisWeek,
		Accuracy: this,
	}
	if totals.ReadThisWeek > 0 && totals.ReadLastWeek > 0 {
		last := accuracy(totals.ReadLastWeek, totals.CorrectedLastWeek)
		summary.AccuracyChange = roundTenth(this - last)
	}

	return summary
}

func accuracy(read, corrected int) float64 {
	if read <= 0 {
		return percent
	}
	share := 1 - float64(min(corrected, read))/float64(read)

	return roundTenth(share * percent)
}

func roundTenth(value float64) float64 {
	return math.Round(value*accuracyPrecision) / accuracyPrecision
}
