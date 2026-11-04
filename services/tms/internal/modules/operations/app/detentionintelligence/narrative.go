package detentionintelligence

import (
	"cmp"
	"math"
	"slices"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

const (
	tagFacility = "f"
	tagCustomer = "c"
	tagSection  = "s"
	tagDanger   = "d"
	tagStrong   = "b"

	maxNextSteps = 4
	// concentrationShare is the share of the hours the summary finds the few facilities for.
	concentrationShare = 0.75
	// maxFocusFacilities caps the few: where detention is spread thin the share is never reached.
	maxFocusFacilities = 10
	// dockShareWorthSaying is when the dock's share of the wait is worth a sentence.
	dockShareWorthSaying = 0.4
	minStepMinutes       = 60
	minNoticeDue         = 5
	noticeOnTimeTarget   = 0.9
	lateShareWorthSaying = 0.3
	oursShareWorthSaying = 0.1
	minArrivalsPerCell   = 5
	earlyShareOfTotal    = 0.15
	minutesPerHour       = 60
	percentScale         = 100
)

var narrativeTags = []string{tagFacility, tagCustomer, tagSection, tagDanger, tagStrong}

type narration struct {
	locale  i18n.Locale
	days    int
	result  *Intelligence
	margins bool
}

type sentenceParts map[string]Segment

func compose(pieces []i18n.RichPiece, parts sentenceParts) []Segment {
	out := make([]Segment, 0, len(pieces))
	for _, piece := range pieces {
		segment, ok := parts[piece.Tag]
		if piece.Tag == "" || !ok {
			segment = Segment{}
		}
		segment.Text = piece.Text
		if segment.Text != "" {
			out = append(out, segment)
		}
	}

	return out
}

func joinSentences(sentences ...[]Segment) []Segment {
	out := make([]Segment, 0, 16)
	for _, sentence := range sentences {
		if len(sentence) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, Segment{Text: " "})
		}
		out = append(out, sentence...)
	}

	return out
}

func strong() Segment { return Segment{Strong: true} }

func facilityRef(f *Facility) Segment {
	return Segment{Target: TargetFacility, TargetID: f.LocationID.String()}
}

func hours(minutes int64) int64 {
	return int64(math.Round(float64(minutes) / minutesPerHour))
}

func percent(share float64) string {
	return strconv.FormatInt(int64(math.Round(share*percentScale)), 10) + "%"
}

func share(part, whole int64) float64 {
	if whole <= 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

func (n *narration) money(amount decimal.Decimal) string {
	return i18n.Translate(
		n.locale,
		"${0}",
		i18n.FormatNumber(n.locale, amount.Abs().Round(0).InexactFloat64()),
	)
}

func (n *narration) number(value int64) string {
	return i18n.FormatNumber(n.locale, float64(value))
}

func (n *narration) hours(minutes int64) string {
	return i18n.FormatNumber(n.locale, float64(hours(minutes)))
}

func (n *narration) duration(minutes float64) string {
	total := int64(math.Round(minutes))
	h, m := total/minutesPerHour, total%minutesPerHour
	switch {
	case h == 0:
		return i18n.Translate(n.locale, "{0}m", m)
	case m == 0:
		return i18n.Translate(n.locale, "{0}h", h)
	default:
		return i18n.Translate(n.locale, "{0}h {1}m", h, m)
	}
}

// summarize is the page's opening briefing, in up to four paragraphs: how long trucks waited
// and whose wait it was; where the schedule guarantees detention and when trucks wait; where the
// money went; and what moved against the window before.
func summarize(n *narration) [][]Segment {
	t := n.result.Totals
	if t.StopCount == 0 {
		return [][]Segment{{{Text: i18n.Translate(
			n.locale,
			"No stop settled detention in the last {0} days.",
			n.days,
		)}}}
	}
	if t.PastFreeMinutes == 0 {
		return [][]Segment{{{Text: i18n.Translate(
			n.locale,
			"No truck waited past its free time in the last {0} days.",
			n.days,
		)}}}
	}

	paragraphs := [][]Segment{
		joinSentences(waitSentence(n), concentrationSentence(n), dockSentence(n), oursSentence(n)),
		joinSentences(freeTimeSentence(n), peakSentence(n)),
		joinSentences(collectedSentence(n), noticeSentence(n), forgivenSentence(n), customerSentence(n)),
		joinSentences(moverSentence(n)),
	}

	out := make([][]Segment, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		if len(paragraph) > 0 {
			out = append(out, paragraph)
		}
	}

	return out
}

func waitSentence(n *narration) []Segment {
	now := hours(n.result.Totals.PastFreeMinutes)
	before := n.result.Previous.PastFreeMinutes
	parts := sentenceParts{tagStrong: strong()}
	if before <= 0 {
		return compose(i18n.TranslateRich(
			n.locale,
			"Trucks waited <b>{0} h</b> past free time in the last {1} days.",
			narrativeTags,
			n.number(now),
			n.days,
		), parts)
	}

	delta := now - hours(before)
	switch {
	case delta > 0:
		return compose(i18n.TranslateRich(
			n.locale,
			"Trucks waited <b>{0} h</b> past free time in the last {1} days, <b>{2} h more</b> than the {1} days before.",
			narrativeTags,
			n.number(now),
			n.days,
			n.number(delta),
		), parts)
	case delta < 0:
		return compose(i18n.TranslateRich(
			n.locale,
			"Trucks waited <b>{0} h</b> past free time in the last {1} days, <b>{2} h less</b> than the {1} days before.",
			narrativeTags,
			n.number(now),
			n.days,
			n.number(-delta),
		), parts)
	default:
		return compose(i18n.TranslateRich(
			n.locale,
			"Trucks waited <b>{0} h</b> past free time in the last {1} days, as long as the {1} days before.",
			narrativeTags,
			n.number(now),
			n.days,
		), parts)
	}
}

// concentrationCut is how many of the ranked facilities it takes to reach the share of the hours.
func concentrationCut(facilities []Facility, total int64) (int, float64) {
	var acc int64
	for idx := range facilities {
		acc += facilities[idx].PastFreeMinutes
		if share(acc, total) >= concentrationShare || idx+1 >= maxFocusFacilities {
			return idx + 1, share(acc, total)
		}
	}

	return len(facilities), share(acc, total)
}

func concentrationSentence(n *narration) []Segment {
	facilities := n.result.Facilities
	const minFacilities = 4
	if len(facilities) < minFacilities {
		return nil
	}
	count, reached := concentrationCut(facilities, n.result.Totals.PastFreeMinutes)
	if count >= len(facilities) {
		return nil
	}
	parts := sentenceParts{
		tagSection: {Target: TargetSection, TargetID: SectionFocus},
		tagStrong:  strong(),
	}
	if n.result.Truncated {
		return compose(i18n.TranslateRich(
			n.locale,
			"<s>{0} facilities</s> account for <b>{1}</b> of it.",
			narrativeTags,
			count,
			percent(reached),
		), parts)
	}

	return compose(i18n.TranslateRich(
		n.locale,
		"<s>{0} of {1} facilities</s> account for <b>{2}</b> of it.",
		narrativeTags,
		count,
		len(facilities),
		percent(reached),
	), parts)
}

func dockSentence(n *narration) []Segment {
	t := n.result.Totals
	dock := share(t.DockMinutes, t.PastFreeMinutes)
	if dock < dockShareWorthSaying {
		return nil
	}

	return compose(i18n.TranslateRich(
		n.locale,
		"On <b>{0}</b> of the hours the truck was on time, so the wait was the dock's.",
		narrativeTags,
		percent(dock),
	), sentenceParts{tagStrong: strong()})
}

func noticeSentence(n *narration) []Segment {
	lost := n.result.Recovery.stepAmount(detentionports.RecoveryNotice)
	if !lost.IsPositive() {
		return nil
	}

	return compose(i18n.TranslateRich(
		n.locale,
		"Another <d>{0}</d> never reached an invoice because notice went out late.",
		narrativeTags,
		n.money(lost),
	), sentenceParts{tagDanger: {Target: TargetSection, TargetID: SectionPaid, Tone: toneDanger}})
}

// oursSentence says how much of the wait the carrier caused: early arrivals waiting for a slot
// and late ones that forfeited free time are fixed in dispatch, not at the dock.
func oursSentence(n *narration) []Segment {
	t := n.result.Totals
	ours := share(t.EarlyMinutes+t.LateMinutes, t.PastFreeMinutes)
	if ours < oursShareWorthSaying {
		return nil
	}

	return compose(i18n.TranslateRich(
		n.locale,
		"<b>{0}</b> is ours to fix: trucks that arrived early and waited for their slot, or late and forfeited their free time.",
		narrativeTags,
		percent(ours),
	), sentenceParts{tagStrong: strong()})
}

// freeTimeSentence names the facilities where the median stop outlasts the free time granted:
// detention there is built into the schedule before anyone is early or late.
func freeTimeSentence(n *narration) []Segment {
	count := 0
	var worst *Facility
	var worstGap float64
	for idx := range n.result.Facilities {
		f := &n.result.Facilities[idx]
		gap := f.MedianDwellMinutes - f.MedianFreeMinutes
		if f.PastFreeMinutes < minStepMinutes || f.MedianFreeMinutes <= 0 || gap <= 0 {
			continue
		}
		count++
		if gap > worstGap {
			worst, worstGap = f, gap
		}
	}
	if worst == nil {
		return nil
	}

	return compose(i18n.TranslateRich(
		n.locale,
		"At <b>{0, plural, one {# facility} other {# facilities}}</b> the median stop runs past the free time granted, so detention is built into the schedule; <f>{1}</f> grants {2} for a {3} stop.",
		narrativeTags,
		count,
		worst.LocationName,
		n.duration(worst.MedianFreeMinutes),
		n.duration(worst.MedianDwellMinutes),
	), sentenceParts{tagStrong: strong(), tagFacility: facilityRef(worst)})
}

// arrivalBands are the hours dispatch books in, the same bands the page's arrival grid draws.
var arrivalBands = [...]struct{ from, to int }{
	{0, 6}, {6, 8}, {8, 10}, {10, 12}, {12, 14}, {14, 16}, {16, 18}, {18, 24},
}

func (n *narration) bandLabel(band int) string {
	switch band {
	case 0:
		return i18n.Translate(n.locale, "12–6a")
	case 1:
		return i18n.Translate(n.locale, "6–8a")
	case 2:
		return i18n.Translate(n.locale, "8–10a")
	case 3:
		return i18n.Translate(n.locale, "10a–12p")
	case 4:
		return i18n.Translate(n.locale, "12–2p")
	case 5:
		return i18n.Translate(n.locale, "2–4p")
	case 6:
		return i18n.Translate(n.locale, "4–6p")
	default:
		return i18n.Translate(n.locale, "6p–12a")
	}
}

func (n *narration) weekdayLabel(isoWeekday int) string {
	switch isoWeekday {
	case 1:
		return i18n.Translate(n.locale, "Mon")
	case 2:
		return i18n.Translate(n.locale, "Tue")
	case 3:
		return i18n.Translate(n.locale, "Wed")
	case 4:
		return i18n.Translate(n.locale, "Thu")
	case 5:
		return i18n.Translate(n.locale, "Fri")
	case 6:
		return i18n.Translate(n.locale, "Sat")
	default:
		return i18n.Translate(n.locale, "Sun")
	}
}

// peakSentence names the weekday and hour band whose arrivals are detained most often, against
// the afternoon, so dispatch can book away from it.
func peakSentence(n *narration) []Segment {
	type cell struct{ stops, detained int }
	var grid [8][len(arrivalBands)]cell
	for _, arrival := range n.result.Arrivals {
		if arrival.Weekday < 1 || arrival.Weekday > 7 {
			continue
		}
		for band, b := range arrivalBands {
			if arrival.Hour >= b.from && arrival.Hour < b.to {
				grid[arrival.Weekday][band].stops += arrival.StopCount
				grid[arrival.Weekday][band].detained += arrival.DetainedCount
			}
		}
	}

	peakDay, peakBand, peakShare := 0, 0, 0.0
	afternoonStops, afternoonDetained := 0, 0
	for day := 1; day <= 7; day++ {
		for band := range arrivalBands {
			c := grid[day][band]
			if band >= 4 && band <= 6 {
				afternoonStops += c.stops
				afternoonDetained += c.detained
			}
			if c.stops < minArrivalsPerCell {
				continue
			}
			if rate := share(int64(c.detained), int64(c.stops)); rate > peakShare {
				peakDay, peakBand, peakShare = day, band, rate
			}
		}
	}
	if peakDay == 0 {
		return nil
	}

	return compose(i18n.TranslateRich(
		n.locale,
		"Trucks arriving <b>{0} {1}</b> are detained <b>{2}</b> of the time, against {3} after noon.",
		narrativeTags,
		n.weekdayLabel(peakDay),
		n.bandLabel(peakBand),
		percent(peakShare),
		percent(share(int64(afternoonDetained), int64(afternoonStops))),
	), sentenceParts{tagStrong: strong()})
}

// collectedSentence follows the money from what the clock accrued to what was paid.
func collectedSentence(n *narration) []Segment {
	r := n.result.Recovery
	if !r.Accrued.IsPositive() {
		return nil
	}

	return compose(i18n.TranslateRich(
		n.locale,
		"Of <b>{0}</b> the clock accrued, <s>{1} was billed</s> and <b>{2}</b> has been collected.",
		narrativeTags,
		n.money(r.Accrued),
		n.money(r.Billed),
		n.money(r.Collected),
	), sentenceParts{
		tagStrong:  strong(),
		tagSection: {Target: TargetSection, TargetID: SectionPaid},
	})
}

func (n *narration) reasonLabel(reason string) string {
	switch detention.WaiverReason(reason) {
	case detention.WaiverReasonWeather:
		return i18n.Translate(n.locale, "Weather")
	case detention.WaiverReasonFacilityClosure:
		return i18n.Translate(n.locale, "Facility closure")
	case detention.WaiverReasonCarrierFault:
		return i18n.Translate(n.locale, "Carrier fault")
	case detention.WaiverReasonEquipmentIssue:
		return i18n.Translate(n.locale, "Equipment issue")
	case detention.WaiverReasonCustomerGoodwill:
		return i18n.Translate(n.locale, "Customer goodwill")
	case detention.WaiverReasonDataCorrection:
		return i18n.Translate(n.locale, "Data correction")
	case detention.WaiverReasonForceMajeure:
		return i18n.Translate(n.locale, "Force majeure")
	default:
		return i18n.Translate(n.locale, "Other")
	}
}

// forgivenSentence names the reason most of the forgiven money was waived for.
func forgivenSentence(n *narration) []Segment {
	waivers := n.result.Waivers
	if len(waivers) == 0 {
		return nil
	}
	total := decimal.Zero
	top := waivers[0]
	for _, waiver := range waivers {
		total = total.Add(waiver.WaivedAmount)
		if waiver.WaivedAmount.GreaterThan(top.WaivedAmount) {
			top = waiver
		}
	}
	if !total.IsPositive() {
		return nil
	}

	return compose(i18n.TranslateRich(
		n.locale,
		"<b>{0}</b> is {1} of the {2} forgiven.",
		narrativeTags,
		n.reasonLabel(top.Reason),
		percent(top.WaivedAmount.Div(total).InexactFloat64()),
		n.money(total),
	), sentenceParts{tagStrong: strong()})
}

// customerSentence names the customer whose free time costs the most in driver pay. It reads
// the margin, so a reader without the margins group never gets it.
func customerSentence(n *narration) []Segment {
	if !n.margins {
		return nil
	}
	var worst *int
	for idx := range n.result.Customers {
		c := &n.result.Customers[idx]
		if !c.NetMargin.Valid || !c.NetMargin.Decimal.IsNegative() {
			continue
		}
		if worst == nil || c.NetMargin.Decimal.LessThan(n.result.Customers[*worst].NetMargin.Decimal) {
			i := idx
			worst = &i
		}
	}
	if worst == nil {
		return nil
	}
	c := &n.result.Customers[*worst]

	return compose(i18n.TranslateRich(
		n.locale,
		"<c>{0}</c> costs <b>{1}</b> more in driver pay than it is billed.",
		narrativeTags,
		c.CustomerName,
		n.money(c.NetMargin.Decimal),
	), sentenceParts{
		tagStrong:   strong(),
		tagCustomer: {Target: TargetCustomer, TargetID: c.CustomerID.String(), Tone: toneDanger},
	})
}

// moverSentence names the facility whose hours past free time moved most against the window
// before, and the policy change behind it when there was one.
func moverSentence(n *narration) []Segment {
	var mover *Facility
	var moved int64
	for idx := range n.result.Facilities {
		f := &n.result.Facilities[idx]
		if f.PreviousPastFreeMinutes == nil {
			continue
		}
		delta := f.PastFreeMinutes - *f.PreviousPastFreeMinutes
		if absMinutes(delta) > absMinutes(moved) {
			mover, moved = f, delta
		}
	}
	if mover == nil || absMinutes(moved) < minStepMinutes {
		return nil
	}

	var change *PolicyChange
	for idx := range n.result.PolicyChanges {
		if n.result.PolicyChanges[idx].LocationID == mover.LocationID {
			change = &n.result.PolicyChanges[idx]
		}
	}
	parts := sentenceParts{tagStrong: strong(), tagFacility: facilityRef(mover)}
	if moved < 0 {
		parts[tagStrong] = Segment{Strong: true, Tone: toneGood}
	} else {
		parts[tagStrong] = Segment{Strong: true, Tone: toneDanger}
	}

	switch {
	case moved > 0 && change != nil:
		return compose(i18n.TranslateRich(
			n.locale,
			"<f>{0}</f> moved most: <b>{1} h more</b> past free time than the window before, after it moved to {2}.",
			narrativeTags, mover.LocationName, n.hours(moved), change.PolicyName,
		), parts)
	case moved > 0:
		return compose(i18n.TranslateRich(
			n.locale,
			"<f>{0}</f> moved most: <b>{1} h more</b> past free time than the window before.",
			narrativeTags, mover.LocationName, n.hours(moved),
		), parts)
	case change != nil:
		return compose(i18n.TranslateRich(
			n.locale,
			"<f>{0}</f> moved most: <b>{1} h less</b> past free time than the window before, after it moved to {2}.",
			narrativeTags, mover.LocationName, n.hours(-moved), change.PolicyName,
		), parts)
	default:
		return compose(i18n.TranslateRich(
			n.locale,
			"<f>{0}</f> moved most: <b>{1} h less</b> past free time than the window before.",
			narrativeTags, mover.LocationName, n.hours(-moved),
		), parts)
	}
}

func absMinutes(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func (r Recovery) stepAmount(bucket string) decimal.Decimal {
	for _, step := range r.Steps {
		if step.Bucket == bucket {
			return step.Amount
		}
	}

	return decimal.Zero
}

type candidate struct {
	step  NextStep
	value float64
}

// nextSteps proposes at most four moves, each sized by what it would recover. Hours and money
// are ranked on one scale by pricing a minute past free time at what the window billed per
// minute; with nothing billed, minutes rank as they stand.
func nextSteps(n *narration) []NextStep {
	t := n.result.Totals
	perMinute := 1.0
	if t.PastFreeMinutes > 0 && t.BilledAmount.IsPositive() {
		perMinute = t.BilledAmount.InexactFloat64() / float64(t.PastFreeMinutes)
	}
	minutesValue := func(minutes int64) float64 { return float64(minutes) * perMinute }

	candidates := make([]candidate, 0, 6)
	add := func(step *NextStep, value float64) {
		if step != nil {
			candidates = append(candidates, candidate{step: *step, value: value})
		}
	}

	if step, minutes := shipperReview(n); step != nil {
		add(step, minutesValue(minutes))
	}
	if step, minutes := freeTimeStep(n); step != nil {
		add(step, minutesValue(minutes))
	}
	if step, amount := noticeStep(n); step != nil {
		add(step, amount)
	}
	if n.margins {
		if step, amount := customerTermsStep(n); step != nil {
			add(step, amount)
		}
	}
	if step, minutes := lateArrivalsStep(n); step != nil {
		add(step, minutesValue(minutes))
	}
	if step, minutes := earlyArrivalsStep(n); step != nil {
		add(step, minutesValue(minutes))
	}

	slices.SortStableFunc(candidates, func(a, b candidate) int { return cmp.Compare(b.value, a.value) })
	out := make([]NextStep, 0, maxNextSteps)
	for _, c := range candidates {
		if len(out) == maxNextSteps {
			break
		}
		out = append(out, c.step)
	}

	return out
}

func minutesImpact(minutes int64) *int64 { return &minutes }

func amountImpact(amount decimal.Decimal) *decimal.Decimal { return &amount }

func maxFacility(facilities []Facility, score func(*Facility) int64) (*Facility, int64) {
	var best *Facility
	var bestScore int64
	for idx := range facilities {
		if s := score(&facilities[idx]); s > bestScore {
			best, bestScore = &facilities[idx], s
		}
	}

	return best, bestScore
}

func shipperReview(n *narration) (*NextStep, int64) {
	f, dock := maxFacility(n.result.Facilities, func(f *Facility) int64 { return f.DockMinutes })
	if f == nil || dock < minStepMinutes {
		return nil, 0
	}

	return &NextStep{
		Kind: StepShipperReview,
		Title: compose(i18n.TranslateRich(
			n.locale,
			"Take the scorecard to <f>{0}</f>",
			narrativeTags,
			f.LocationName,
		), sentenceParts{tagFacility: facilityRef(f)}),
		Detail: i18n.Translate(
			n.locale,
			"{0} h of waiting with the truck on time for its appointment. Ask for later appointments, more free time or drop trailers.",
			n.hours(dock),
		),
		ImpactMinutes: minutesImpact(dock),
		ImpactCaption: i18n.Translate(n.locale, "on-time waiting"),
		LocationID:    f.LocationID,
	}, dock
}

func freeTimeStep(n *narration) (*NextStep, int64) {
	f, _ := maxFacility(n.result.Facilities, func(f *Facility) int64 {
		if f.PastFreeMinutes < minStepMinutes {
			return 0
		}
		return int64(f.MedianDwellMinutes - f.MedianFreeMinutes)
	})
	if f == nil {
		return nil, 0
	}

	return &NextStep{
		Kind: StepFreeTime,
		Title: compose(i18n.TranslateRich(
			n.locale,
			"Free time at <f>{0}</f> is shorter than its typical stop",
			narrativeTags,
			f.LocationName,
		), sentenceParts{tagFacility: facilityRef(f)}),
		Detail: i18n.Translate(
			n.locale,
			"{0} of free time against a {1} median stop, so most stops are detained before anyone is late.",
			n.duration(f.MedianFreeMinutes),
			n.duration(f.MedianDwellMinutes),
		),
		ImpactMinutes: minutesImpact(f.PastFreeMinutes),
		ImpactCaption: i18n.Translate(n.locale, "past free time there"),
		LocationID:    f.LocationID,
	}, f.PastFreeMinutes
}

func noticeStep(n *narration) (*NextStep, float64) {
	lost := n.result.Recovery.stepAmount(detentionports.RecoveryNotice)
	if !lost.IsPositive() {
		return nil, 0
	}
	held := 0
	for _, step := range n.result.Recovery.Steps {
		if step.Bucket == detentionports.RecoveryNotice {
			held = step.OccurrenceCount
		}
	}

	var worst *Facility
	worstRate := noticeOnTimeTarget
	for idx := range n.result.Facilities {
		f := &n.result.Facilities[idx]
		if f.NoticeDueCount < minNoticeDue {
			continue
		}
		if rate := share(int64(f.NoticeOnTimeCount), int64(f.NoticeDueCount)); rate < worstRate {
			worst, worstRate = f, rate
		}
	}

	step := &NextStep{
		Kind:          StepNotice,
		ImpactAmount:  amountImpact(lost),
		ImpactCaption: i18n.Translate(n.locale, "lost to late notice"),
	}
	if worst == nil {
		step.Title = []Segment{{Text: i18n.Translate(n.locale, "Send notice before the deadline")}}
		step.Detail = i18n.Translate(
			n.locale,
			"{0, plural, one {# charge was held} other {# charges were held}} because notice missed its deadline.",
			held,
		)
	} else {
		step.Title = compose(i18n.TranslateRich(
			n.locale,
			"Send notice before the deadline at <f>{0}</f>",
			narrativeTags,
			worst.LocationName,
		), sentenceParts{tagFacility: facilityRef(worst)})
		step.Detail = i18n.Translate(
			n.locale,
			"{0, plural, one {# charge was held} other {# charges were held}} because notice missed its deadline. Notice goes out on time on {1} of stops there.",
			held,
			percent(worstRate),
		)
		step.LocationID = worst.LocationID
	}

	return step, lost.InexactFloat64()
}

func customerTermsStep(n *narration) (*NextStep, float64) {
	var worst *int
	for idx := range n.result.Customers {
		c := &n.result.Customers[idx]
		if !c.NetMargin.Valid || !c.NetMargin.Decimal.IsNegative() {
			continue
		}
		if worst == nil || c.NetMargin.Decimal.LessThan(n.result.Customers[*worst].NetMargin.Decimal) {
			i := idx
			worst = &i
		}
	}
	if worst == nil {
		return nil, 0
	}
	c := &n.result.Customers[*worst]
	loss := c.NetMargin.Decimal

	return &NextStep{
		Kind: StepCustomerTerms,
		Title: compose(i18n.TranslateRich(
			n.locale,
			"Free time for <c>{0}</c> costs more than it bills",
			narrativeTags,
			c.CustomerName,
		), sentenceParts{tagCustomer: {Target: TargetCustomer, TargetID: c.CustomerID.String()}}),
		Detail: i18n.Translate(
			n.locale,
			"Driver pay ran {0} past billed detention over {1, plural, one {# stop} other {# stops}}. Raise the free time or the detention rate at renewal.",
			n.money(loss),
			c.StopCount,
		),
		ImpactAmount:  amountImpact(loss),
		ImpactCaption: i18n.Translate(n.locale, "net this window"),
		CustomerID:    c.CustomerID,
	}, loss.Abs().InexactFloat64()
}

func lateArrivalsStep(n *narration) (*NextStep, int64) {
	f, late := maxFacility(n.result.Facilities, func(f *Facility) int64 {
		if share(f.LateMinutes, f.PastFreeMinutes) < lateShareWorthSaying {
			return 0
		}
		return f.LateMinutes
	})
	if f == nil || late < minStepMinutes {
		return nil, 0
	}

	return &NextStep{
		Kind: StepLateArrivals,
		Title: compose(i18n.TranslateRich(
			n.locale,
			"Trucks reach <f>{0}</f> late",
			narrativeTags,
			f.LocationName,
		), sentenceParts{tagFacility: facilityRef(f)}),
		Detail: i18n.Translate(
			n.locale,
			"{0} of its {1} h past free time followed a missed appointment, so the free time was forfeited.",
			n.hours(late),
			n.hours(f.PastFreeMinutes),
		),
		ImpactMinutes: minutesImpact(late),
		ImpactCaption: i18n.Translate(n.locale, "after late arrivals"),
		LocationID:    f.LocationID,
	}, late
}

func earlyArrivalsStep(n *narration) (*NextStep, int64) {
	t := n.result.Totals
	if share(t.EarlyMinutes, t.PastFreeMinutes) < earlyShareOfTotal {
		return nil, 0
	}
	f, early := maxFacility(n.result.Facilities, func(f *Facility) int64 { return f.EarlyMinutes })
	if f == nil || early < minStepMinutes {
		return nil, 0
	}

	return &NextStep{
		Kind: StepEarlyArrivals,
		Title: compose(i18n.TranslateRich(
			n.locale,
			"Dispatch to the appointment at <f>{0}</f>, not ahead of it",
			narrativeTags,
			f.LocationName,
		), sentenceParts{tagFacility: facilityRef(f)}),
		Detail: i18n.Translate(
			n.locale,
			"{0} h went to waiting for a slot after arriving early.",
			n.hours(early),
		),
		ImpactMinutes: minutesImpact(early),
		ImpactCaption: i18n.Translate(n.locale, "waiting for a slot"),
		LocationID:    f.LocationID,
	}, early
}
