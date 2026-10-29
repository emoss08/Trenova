package factoringvolume

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger/factoringledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness/factoringreadinessdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance/factoringremittancedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/seeds/volume/ledgervolume"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/pkg/factoringremit"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

// -- Factoring phase --

type facInvoice struct {
	rec        *volumegen.ShipmentRecord
	inv        *invoice.Invoice
	factor     *volumegen.Factor
	readyAt    int64
	face       int64
	items      []*facItem
	rejectedAt int64
	updatedAt  int64
	touched    bool
}

type facItem struct {
	fi           *facInvoice
	sub          *facSub
	item         *factoringsubmission.SubmissionItem
	fundOn       int64
	payOn        int64
	chargebackOn int64
	reversalOn   int64
	decline      string
	extraFee     int64
	chargeback   *factoringledger.Movement
	chargebackID pulid.ID
}

type facSub struct {
	factor *volumegen.Factor
	sub    *factoringsubmission.Submission
	items  []*facItem
	slot   int64
	keys   []string
}

type facEvent struct {
	at   int64
	kind factoringremittance.RemittanceLineKind
	item *facItem
	cash int64
}

type facAppliedLine struct {
	line    *factoringremittance.RemittanceLine
	entryID pulid.ID
}

type facRun struct {
	setup       *ledgervolume.Setup
	rows        *ledgervolume.Rows
	accounts    map[pulid.ID]factoring.PostingAccounts
	bookable    map[pulid.ID]bool
	invoices    []*facInvoice
	subs        []*facSub
	entries     []*factoringledger.ReserveEntry
	remittances []*factoringremittance.Remittance
	lines       []*factoringremittance.RemittanceLine
	applied     []facAppliedLine
	profiles    []*factoringremittance.RemittanceProfile
	docs        []*document.Document
	states      []*factoringreadiness.InvoiceState
	remStamps   []volumegen.Stamp
	lineStamps  []volumegen.Stamp
	excluded    int
}

//nolint:gocognit,funlen // the phase is a sequence of steps over one timeline
func (g *factoringPhase) seedFactoring(ctx context.Context) error {
	st := g.Factoring
	if st == nil || len(st.Factors) == 0 {
		return nil
	}
	setup, err := g.LoadSetup(ctx)
	if err != nil {
		return err
	}
	run := &facRun{
		setup: setup,
		rows: &ledgervolume.Rows{
			Balances: make(map[ledgervolume.BalanceKey]*ledgervolume.Balance),
		},
		accounts: make(map[pulid.ID]factoring.PostingAccounts, len(st.Factors)),
		bookable: make(map[pulid.ID]bool, len(st.Factors)),
	}
	for _, f := range st.Factors {
		accounts := f.Agreement.PostingAccounts(setup.AR, setup.Cash)
		run.accounts[f.Company.ID] = accounts
		run.bookable[f.Company.ID] = len(setup.Periods) > 0 &&
			len(accounts.Missing(f.Agreement.AccountingTreatment)) == 0
		if !run.bookable[f.Company.ID] {
			g.Opts.Logf("  ! %s: no open period or posting accounts; funding is not booked",
				f.Company.Name)
		}
	}

	g.facCollect(st, run)
	if len(run.invoices) == 0 {
		return nil
	}
	if err = g.facSubmit(ctx, run); err != nil {
		return err
	}
	g.facPlanEvents(run)
	if err = g.facRemit(run); err != nil {
		return err
	}
	if err = g.facExclude(run); err != nil {
		return err
	}
	g.facStates(run)

	if len(run.rows.Journals) > 0 {
		if err = g.Materialize(ctx, run.rows); err != nil {
			return err
		}
	}
	if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		return g.facWrite(ctx, tx, run)
	}); err != nil {
		return err
	}
	g.facCount(run)

	return nil
}

func (g *factoringPhase) facCollect(st *volumegen.FactorState, run *facRun) {
	now := volumegen.Unix(g.Now)
	for _, r := range g.Shipments {
		inv := r.Invoice
		if inv == nil || inv.Status == invoice.StatusDraft || inv.IsCreditMemo() {
			continue
		}
		var factor *volumegen.Factor
		if inv.FactoringCompanyID != nil {
			factor = st.ByID[*inv.FactoringCompanyID]
		}
		if factor == nil && !inv.FactoringOverride.Excludes() {
			continue
		}
		fi := &facInvoice{rec: r, inv: inv, factor: factor, face: inv.OpenBalanceMinor()}
		run.invoices = append(run.invoices, fi)
		if factor == nil || inv.Status != invoice.StatusPosted || fi.face <= 0 ||
			inv.DisputeStatus == invoice.DisputeStatusDisputed {
			continue
		}
		noa := factor.Noas[inv.CustomerID]
		if noa == nil || noa.AcknowledgedAt == nil {
			continue
		}
		ready := max(
			*inv.PostedAt+g.Fake.Between64(5*volumegen.Minute, 2*volumegen.Hour),
			*noa.AcknowledgedAt,
		)
		if !factor.Agreement.InForce(ready) ||
			(noa.ExpiresAt != nil && ready >= *noa.ExpiresAt) ||
			(noa.ReleasedAt != nil && ready >= *noa.ReleasedAt) || ready > now {
			continue
		}
		fi.readyAt = ready
	}
}

// facSlot is the next time the organization sends schedules: the agreement's
// submission time on the next business day at or after at.
func facSlot(at int64, minuteOfDay int) int64 {
	t := time.Unix(at, 0).UTC()
	slot := time.Date(t.Year(), t.Month(), t.Day(), 0, minuteOfDay, 0, 0, time.UTC)
	if !slot.After(t) {
		slot = slot.AddDate(0, 0, 1)
	}
	for slot.Weekday() == time.Saturday || slot.Weekday() == time.Sunday {
		slot = slot.AddDate(0, 0, 1)
	}
	return slot.Unix()
}

func facBusinessDays(at int64, days int) int64 {
	t := time.Unix(at, 0).UTC()
	for days > 0 {
		t = t.AddDate(0, 0, 1)
		if t.Weekday() != time.Saturday && t.Weekday() != time.Sunday {
			days--
		}
	}
	return t.Unix()
}

func facWeekStart(at int64) int64 {
	t := time.Unix(at, 0).UTC()
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset).Unix()
}

//nolint:gocognit,funlen // scheduling, delivery and resubmission share one timeline
func (g *factoringPhase) facSubmit(ctx context.Context, run *facRun) error {
	now := volumegen.Unix(g.Now)
	type slotKey struct {
		factor pulid.ID
		slot   int64
	}
	groups := make(map[slotKey][]*facInvoice)
	for _, fi := range run.invoices {
		if fi.readyAt == 0 {
			continue
		}
		slot := facSlot(fi.readyAt, fi.factor.Agreement.SubmissionTime)
		switch g.Fake.Weighted([]int{70, 20, 10}) {
		case 1:
			slot = facSlot(slot, fi.factor.Agreement.SubmissionTime)
		case 2:
			slot = facSlot(
				facBusinessDays(slot, g.Fake.Between(1, 2)),
				fi.factor.Agreement.SubmissionTime,
			)
		}
		noa := fi.factor.Noas[fi.inv.CustomerID]
		if slot+10*volumegen.Minute > now || (noa.ReleasedAt != nil && slot >= *noa.ReleasedAt) {
			continue
		}
		key := slotKey{factor: fi.factor.Company.ID, slot: slot}
		groups[key] = append(groups[key], fi)
	}

	keys := make([]slotKey, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].slot != keys[j].slot {
			return keys[i].slot < keys[j].slot
		}
		return keys[i].factor < keys[j].factor
	})

	byFactor := make(map[pulid.ID]*volumegen.Factor, len(run.bookable))
	for _, fi := range run.invoices {
		if fi.factor != nil {
			byFactor[fi.factor.Company.ID] = fi.factor
		}
	}
	var resubmit []*facInvoice
	var resubmitAt []int64
	for _, key := range keys {
		members := groups[key]
		for begin := 0; begin < len(members); begin += factoringsubmission.MaxSubmissionInvoices {
			batch := members[begin:min(begin+factoringsubmission.MaxSubmissionInvoices, len(members))]
			fs := g.facNewSub(byFactor[key.factor], key.slot, batch)
			run.subs = append(run.subs, fs)
			if cancelled := g.facDeliver(fs, now); cancelled > 0 {
				for _, it := range fs.items {
					resubmit = append(resubmit, it.fi)
					resubmitAt = append(resubmitAt, cancelled)
				}
			}
		}
	}

	type retryKey struct {
		factor pulid.ID
		slot   int64
	}
	retries := make(map[retryKey][]*facInvoice)
	for i, fi := range resubmit {
		// Readiness clears ready_since while an invoice is on a schedule and
		// sets it afresh on release, so a resubmission keys on the release time.
		fi.readyAt = resubmitAt[i]
		slot := facSlot(resubmitAt[i], fi.factor.Agreement.SubmissionTime)
		if slot+10*volumegen.Minute > now {
			continue
		}
		key := retryKey{factor: fi.factor.Company.ID, slot: slot}
		retries[key] = append(retries[key], fi)
	}
	retryKeys := make([]retryKey, 0, len(retries))
	for key := range retries {
		retryKeys = append(retryKeys, key)
	}
	sort.Slice(retryKeys, func(i, j int) bool {
		if retryKeys[i].slot != retryKeys[j].slot {
			return retryKeys[i].slot < retryKeys[j].slot
		}
		return retryKeys[i].factor < retryKeys[j].factor
	})
	for _, key := range retryKeys {
		members := retries[key]
		fs := g.facNewSub(byFactor[key.factor], key.slot, members)
		fs.sub.Status = factoringsubmission.SubmissionStatusSent
		fs.sub.Attempts = 1
		fs.sub.SentAt = volumegen.Ptr(
			key.slot + g.Fake.Between64(volumegen.Minute, 8*volumegen.Minute),
		)
		fs.sub.TransportReference = g.facMessageID(fs.factor)
		run.subs = append(run.subs, fs)
	}
	sort.SliceStable(run.subs, func(i, j int) bool { return run.subs[i].slot < run.subs[j].slot })

	slots := make([]volumegen.NumberSlot, len(run.subs))
	for i, fs := range run.subs {
		slots[i] = volumegen.NumberSlot{At: fs.slot}
	}
	numbers, err := g.Numbers.Allocate(ctx, sequenceconfig.SequenceTypeFactoringSubmission, slots)
	if err != nil {
		return err
	}
	for i, fs := range run.subs {
		fs.sub.Number = numbers[i]
	}
	return nil
}

func (g *factoringPhase) facNewSub(f *volumegen.Factor, slot int64, members []*facInvoice) *facSub {
	submitter := g.PickUser(g.Refs.UserIDs)
	fs := &facSub{factor: f, slot: slot, keys: []string{f.Company.ID.String()}}
	fs.sub = &factoringsubmission.Submission{
		ID:                 volumegen.IDAt("facsub_", slot),
		OrganizationID:     g.Refs.OrgID,
		BusinessUnitID:     g.Refs.BuID,
		FactoringCompanyID: f.Company.ID,
		AgreementID:        f.Agreement.ID,
		Status:             factoringsubmission.SubmissionStatusSending,
		CurrencyCode:       money.DefaultCurrencyCode,
		TransportMethod:    f.Transport.Method,
		TransportTarget:    f.Transport.Target(),
		SendAttemptGroup:   1,
		SubmittedByID:      submitter,
		CreatedAt:          slot,
		UpdatedAt:          slot,
	}
	for i, fi := range members {
		funding := factoring.ExpectedFunding(
			fi.face,
			f.Agreement,
			factoring.DaysToRun(fi.inv.DueDate, slot),
		)
		item := &factoringsubmission.SubmissionItem{
			ID:                   volumegen.IDAt("facsit_", slot),
			OrganizationID:       g.Refs.OrgID,
			BusinessUnitID:       g.Refs.BuID,
			SubmissionID:         fs.sub.ID,
			InvoiceID:            fi.inv.ID,
			InvoiceNumber:        fi.inv.Number,
			CustomerID:           fi.inv.CustomerID,
			Position:             i + 1,
			FaceMinor:            funding.FaceMinor,
			ExpectedAdvanceMinor: funding.AdvanceMinor,
			ExpectedFeeMinor:     funding.FeeMinor,
			ExpectedReserveMinor: funding.ReserveMinor,
			ExpectedDays:         funding.Days,
			Status:               factoringsubmission.SubmissionItemStatusPending,
			CreatedAt:            slot,
			UpdatedAt:            slot,
		}
		it := &facItem{fi: fi, sub: fs, item: item}
		fi.items = append(fi.items, it)
		fs.items = append(fs.items, it)
		fs.sub.Items = append(fs.sub.Items, item)
		fs.keys = append(fs.keys, fi.inv.ID.String()+"@"+strconv.FormatInt(fi.readyAt, 10))
	}
	fs.sub.Totalize()
	fs.sub.IdempotencyKey = facSubmissionKey(fs.keys)
	return fs
}

// facSubmissionKey is factoringsubmissionservice.submissionKey: the factor, then the
// sorted invoices with the moment each became ready.
func facSubmissionKey(parts []string) string {
	sorted := slices.Clone(parts)
	slices.Sort(sorted[1:])
	sum := sha256.Sum256([]byte(strings.Join(sorted, "|")))
	return "facsub-" + hex.EncodeToString(sum[:16])
}

// facDeliver decides how sending the schedule went and returns the time it
// was cancelled, if it was.
func (g *factoringPhase) facDeliver(fs *facSub, now int64) int64 {
	sub := fs.sub
	sentAt := fs.slot + g.Fake.Between64(volumegen.Minute, 8*volumegen.Minute)
	if !g.Fake.Chance(0.04) {
		sub.Status = factoringsubmission.SubmissionStatusSent
		sub.Attempts = 1
		sub.SentAt = volumegen.Ptr(sentAt)
		sub.TransportReference = g.facMessageID(fs.factor)
		sub.UpdatedAt = sentAt
		return 0
	}

	failedAt := fs.slot + g.Fake.Between64(2*volumegen.Minute, 20*volumegen.Minute)
	sub.Status = factoringsubmission.SubmissionStatusFailed
	sub.Attempts = 3
	sub.FailedAt = volumegen.Ptr(failedAt)
	sub.LastError = volumegen.Pick(g.Fake, facFailures)
	sub.UpdatedAt = failedAt
	if now-fs.slot < 2*volumegen.Day {
		return 0
	}
	if g.Fake.Chance(0.75) {
		retried := min(
			failedAt+g.Fake.Between64(volumegen.Hour, 20*volumegen.Hour),
			now-volumegen.Minute,
		)
		sub.Status = factoringsubmission.SubmissionStatusSent
		sub.SendAttemptGroup = 2
		sub.Attempts = 1
		sub.SentAt = volumegen.Ptr(retried + 2*volumegen.Minute)
		sub.FailedAt = nil
		sub.TransportReference = g.facMessageID(fs.factor)
		sub.UpdatedAt = retried + 2*volumegen.Minute
		return 0
	}
	cancelled := min(
		failedAt+g.Fake.Between64(2*volumegen.Hour, 30*volumegen.Hour),
		now-volumegen.Minute,
	)
	sub.Status = factoringsubmission.SubmissionStatusCancelled
	sub.CancelledAt = volumegen.Ptr(cancelled)
	sub.CancelReason = "Factor asked for the invoices on a fresh schedule after the failed delivery"
	sub.UpdatedAt = cancelled
	for _, it := range fs.items {
		it.item.Status = factoringsubmission.SubmissionItemStatusReleased
		it.item.ReleasedAt = volumegen.Ptr(cancelled)
		it.item.UpdatedAt = cancelled
	}
	return cancelled
}

func (g *factoringPhase) facMessageID(f *volumegen.Factor) string {
	return fmt.Sprintf(
		"<%s.%s@mail.%s.example.com>",
		g.Fake.Digits(12),
		strings.ToLower(g.Fake.Letters(8)),
		f.Slug,
	)
}

// facPlanEvents decides what the factor does with each schedule's invoices:
// fund or decline them, and later release the reserve when the customer
// pays, or charge the invoice back when it does not.
func (g *factoringPhase) facPlanEvents(run *facRun) {
	for _, fs := range run.subs {
		if fs.sub.Status != factoringsubmission.SubmissionStatusSent || fs.sub.SentAt == nil {
			continue
		}
		agreement := fs.factor.Agreement
		fundOn := facBusinessDays(*fs.sub.SentAt, max(agreement.FundingDays, 0)) +
			g.Fake.Between64(2*volumegen.Hour, 6*volumegen.Hour)
		recourse := agreement.Recourse == factoring.RecourseFull
		for _, it := range fs.items {
			it.fundOn = fundOn
			if g.Fake.Chance(0.06) {
				it.fundOn = facBusinessDays(fundOn, g.Fake.Between(3, 6))
			}
			if g.Fake.Chance(0.03) {
				it.decline = volumegen.Pick(g.Fake, facDeclineReasons)
				continue
			}
			due := it.fi.inv.InvoiceDate + 30*volumegen.Day
			if it.fi.inv.DueDate != nil {
				due = *it.fi.inv.DueDate
			}
			floor := facWeekStart(it.fundOn) + 8*volumegen.Day
			if recourse && g.Fake.Chance(0.025) {
				it.chargebackOn = max(
					it.fundOn+int64(agreement.ChargebackWindowDays)*volumegen.Day,
					floor,
				)
				if g.Fake.Chance(0.2) {
					it.reversalOn = facWeekStart(it.chargebackOn) + 10*volumegen.Day +
						g.Fake.Between64(2, 20)*volumegen.Day
					it.payOn = max(it.reversalOn+g.Fake.Between64(2, 10)*volumegen.Day,
						facWeekStart(it.reversalOn)+8*volumegen.Day)
				}
			} else {
				it.payOn = max(due+g.Fake.Between64(-12, 25)*volumegen.Day, floor)
			}
			if it.payOn > 0 && it.payOn > due+30*volumegen.Day {
				it.extraFee = int64(math.Round(float64(it.item.FaceMinor) * 0.01))
			}
		}
	}
}

type facBucket struct {
	factor *volumegen.Factor
	week   int64
	events []*facEvent
}

//nolint:gocognit,funlen // staging, matching and booking happen in the factor's own order
func (g *factoringPhase) facRemit(run *facRun) error {
	now := volumegen.Unix(g.Now)
	buckets := make(map[string]*facBucket)
	add := func(f *volumegen.Factor, ev *facEvent) {
		week := facWeekStart(ev.at)
		key := f.Company.ID.String() + "|" + strconv.FormatInt(week, 10)
		b, ok := buckets[key]
		if !ok {
			b = &facBucket{factor: f, week: week}
			buckets[key] = b
		}
		b.events = append(b.events, ev)
	}
	reversals := make([]*facItem, 0)
	for _, fs := range run.subs {
		for _, it := range fs.items {
			if it.fundOn == 0 {
				continue
			}
			if it.decline != "" {
				add(
					fs.factor,
					&facEvent{
						at:   it.fundOn,
						kind: factoringremittance.RemittanceLineRejection,
						item: it,
					},
				)
				continue
			}
			add(
				fs.factor,
				&facEvent{at: it.fundOn, kind: factoringremittance.RemittanceLineFunding, item: it},
			)
			if it.chargebackOn > 0 {
				add(
					fs.factor,
					&facEvent{
						at:   it.chargebackOn,
						kind: factoringremittance.RemittanceLineChargeback,
						item: it,
					},
				)
			}
			if it.reversalOn > 0 {
				reversals = append(reversals, it)
			}
			if it.payOn > 0 {
				add(
					fs.factor,
					&facEvent{
						at:   it.payOn,
						kind: factoringremittance.RemittanceLineReserveRelease,
						item: it,
					},
				)
			}
		}
	}

	type action struct {
		at     int64
		bucket *facBucket
		item   *facItem
	}
	actions := make([]action, 0, len(buckets)+len(reversals))
	for _, b := range buckets {
		received := b.week + 7*volumegen.Day + 6*volumegen.Hour + g.Fake.Between64(
			0,
			4*volumegen.Hour,
		)
		if received > now {
			continue
		}
		actions = append(actions, action{at: received, bucket: b})
	}
	for _, it := range reversals {
		if it.reversalOn <= now {
			actions = append(actions, action{at: it.reversalOn, item: it})
		}
	}
	sort.SliceStable(actions, func(i, j int) bool { return actions[i].at < actions[j].at })

	for _, a := range actions {
		if a.bucket != nil {
			if err := g.facApplyBucket(run, a.bucket, a.at); err != nil {
				return err
			}
			continue
		}
		if err := g.facReverse(run, a.item, a.at); err != nil {
			return err
		}
	}

	g.facOddRemittances(run, now)
	return nil
}

var facKindOrder = map[factoringremittance.RemittanceLineKind]int{
	factoringremittance.RemittanceLineFunding:        0,
	factoringremittance.RemittanceLineRejection:      1,
	factoringremittance.RemittanceLineChargeback:     2,
	factoringremittance.RemittanceLineReserveRelease: 3,
	factoringremittance.RemittanceLineOther:          4,
}

func facMoney(minor int64) string {
	return money.DecimalFromMinor(minor).StringFixed(2)
}

//nolint:gocognit,funlen // one remittance file from receipt to booking
func (g *factoringPhase) facApplyBucket(run *facRun, b *facBucket, received int64) error {
	now := volumegen.Unix(g.Now)
	f := b.factor
	sort.SliceStable(b.events, func(i, j int) bool {
		if facKindOrder[b.events[i].kind] != facKindOrder[b.events[j].kind] {
			return facKindOrder[b.events[i].kind] < facKindOrder[b.events[j].kind]
		}
		return b.events[i].at < b.events[j].at
	})
	if g.Fake.Chance(0.4) {
		b.events = append(
			b.events,
			&facEvent{
				at:   b.week + 4*volumegen.Day,
				kind: factoringremittance.RemittanceLineOther,
				cash: -2500,
			},
		)
	}

	weekEnd := b.week + 7*volumegen.Day - volumegen.Day
	fileName := fmt.Sprintf("%s_remittance_%s%s", strings.ToUpper(f.Company.Code),
		time.Unix(weekEnd, 0).UTC().Format("2006-01-02"), f.Style.Ext)
	applyAt := received + g.Fake.Between64(volumegen.Hour, 30*volumegen.Hour)
	applied := applyAt <= now && run.bookable[f.Company.ID]
	uploader := g.PickUser(g.Refs.UserIDs)

	rows := make([][]string, 0, len(b.events))
	for _, ev := range b.events {
		rows = append(rows, g.facCells(f, ev))
	}
	sheet := &rateimport.Sheet{Headers: f.Style.Headers, Rows: rows, FirstDataRow: 2}
	mapping := factoringremit.GuessMapping(f.Style.Headers)
	parsed := factoringremit.ToDomain(factoringremit.Parse(sheet, mapping, factoringremit.Options{
		DefaultKind: factoringremittance.RemittanceLineFunding,
		Location:    time.UTC,
	}))
	if len(parsed) != len(b.events) {
		return fmt.Errorf(
			"remittance %s parsed %d lines from %d rows",
			fileName,
			len(parsed),
			len(b.events),
		)
	}

	rem := &factoringremittance.Remittance{
		ID:                 volumegen.IDAt("facrem_", received),
		OrganizationID:     g.Refs.OrgID,
		BusinessUnitID:     g.Refs.BuID,
		FactoringCompanyID: f.Company.ID,
		FileName:           fileName,
		Source:             factoringremittance.RemittanceSourceUpload,
		SourceFormat:       facFormat(f.Style.Ext),
		Status:             factoringremittance.RemittanceStatusParsed,
		DefaultKind:        factoringremittance.RemittanceLineFunding,
		RemittedOn:         volumegen.Ptr(weekEnd),
		CurrencyCode:       money.DefaultCurrencyCode,
		Mapping:            mapping.Strings(),
		Headers:            f.Style.Headers,
		UnmappedHeaders:    factoringremit.UnmappedHeaders(f.Style.Headers, mapping),
		StagedAt:           volumegen.Ptr(received + 2*volumegen.Minute),
		CreatedByID:        uploader,
	}
	doc := g.DocumentRow(factoringremittance.RemittanceDocumentResourceType, rem.ID, fileName,
		facContentType(f.Style.Ext), document.ProcessingProfileNone, received, uploader,
		g.Fake.Between64(4_000, 60_000))
	rem.DocumentID = doc.ID
	run.docs = append(run.docs, doc)
	for _, line := range parsed {
		line.ID = volumegen.IDAt("facrml_", received)
		line.OrganizationID = g.Refs.OrgID
		line.BusinessUnitID = g.Refs.BuID
		line.RemittanceID = rem.ID
	}
	rem.Lines = parsed

	factoringremittance.MatchRemittance(parsed, g.facCandidates(run, f))

	touched := make(map[*facSub]int64)
	if applied {
		for i, line := range parsed {
			ev := b.events[i]
			if ev.item == nil {
				if line.MatchStatus.IsException() {
					if err := line.Ignore("Bank fee on the factor's wire; booked with the month's bank charges"); err != nil {
						return err
					}
				}
				continue
			}
			if !facBookable(ev) {
				if line.MatchStatus == factoringremittance.RemittanceLineMatched {
					line.Suggest(nil, factoringremittance.MatchReasonNoOpenItem)
				}
				continue
			}
			if line.MatchStatus != factoringremittance.RemittanceLineMatched ||
				line.SubmissionItemID != ev.item.item.ID {
				score, _ := factoringremittance.ScoreRemittanceLine(
					line,
					&factoringremittance.RemittanceCandidate{
						Item:             ev.item.item,
						SubmissionNumber: ev.item.sub.sub.Number,
						SentAt:           ev.item.sub.sub.SentAt,
						Recourse:         f.Agreement.Recourse == factoring.RecourseFull,
					},
				)
				line.MatchTo(
					ev.item.item,
					factoringremittance.RemittanceMatchedByPerson,
					max(score, 1),
					factoringremittance.MatchReasonChosenByPerson,
				)
			}
			entryID, err := g.facBookLine(run, rem, line, ev, applyAt, uploader)
			if err != nil {
				return err
			}
			line.MarkApplied(entryID)
			if ev.kind == factoringremittance.RemittanceLineFunding ||
				ev.kind == factoringremittance.RemittanceLineRejection {
				touched[ev.item.sub] = max(touched[ev.item.sub], ev.at)
			}
			if entryID.IsNotNil() {
				run.applied = append(run.applied, facAppliedLine{line: line, entryID: entryID})
			}
		}
		for fs, fundedOn := range touched {
			fs.sub.Resolve(fundedOn)
			fs.sub.UpdatedAt = applyAt
		}
		rem.MarkApplied(applyAt, uploader)
	} else {
		rem.Recount()
	}

	updated := received + 2*volumegen.Minute
	if rem.AppliedAt != nil {
		updated = *rem.AppliedAt
	}
	run.remittances = append(run.remittances, rem)
	run.remStamps = append(
		run.remStamps,
		volumegen.Stamp{Id: rem.ID, Created: received, Updated: updated},
	)
	for _, line := range parsed {
		run.lines = append(run.lines, line)
		run.lineStamps = append(
			run.lineStamps,
			volumegen.Stamp{Id: line.ID, Created: received, Updated: updated},
		)
	}
	if !f.Profiled {
		f.Profiled = true
		run.profiles = append(run.profiles, &factoringremittance.RemittanceProfile{
			ID:                 volumegen.IDAt("facrmp_", received),
			OrganizationID:     g.Refs.OrgID,
			BusinessUnitID:     g.Refs.BuID,
			FactoringCompanyID: f.Company.ID,
			Mapping:            mapping.Strings(),
			HeaderSignature:    factoringremit.HeaderSignature(f.Style.Headers),
			CreatedAt:          received,
			UpdatedAt:          received,
		})
	}
	return nil
}

// facBookable says whether the line's item is still where the factor's report
// expects it; one that is not stays on the remittance for someone to resolve.
func facBookable(ev *facEvent) bool {
	switch ev.kind {
	case factoringremittance.RemittanceLineFunding, factoringremittance.RemittanceLineRejection:
		return ev.item.item.Status == factoringsubmission.SubmissionItemStatusPending
	case factoringremittance.RemittanceLineReserveRelease,
		factoringremittance.RemittanceLineChargeback:
		return ev.item.item.Status == factoringsubmission.SubmissionItemStatusFunded
	case factoringremittance.RemittanceLineOther:
	}
	return false
}

func facFormat(ext string) factoringremittance.RemittanceFormat {
	switch ext {
	case ".xlsx":
		return factoringremittance.RemittanceFormatXlsx
	case ".pdf":
		return factoringremittance.RemittanceFormatPdf
	default:
		return factoringremittance.RemittanceFormatCsv
	}
}

func facContentType(ext string) string {
	switch ext {
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".pdf":
		return "application/pdf"
	default:
		return "text/csv"
	}
}

// facCells writes one row of the factor's report in its own column layout.
// A few invoice numbers lose their prefix, as factors' systems often print
// them, so those lines land as suggestions for someone to confirm.
func (g *factoringPhase) facCells(f *volumegen.Factor, ev *facEvent) []string {
	date := time.Unix(ev.at, 0).UTC().Format("2006-01-02")
	if ev.item == nil {
		return []string{
			"", "WIRE-" + g.Fake.Digits(6), f.Company.Name, f.Style.Other,
			"", "", "", "", facMoney(ev.cash), date,
		}
	}
	it := ev.item
	number := it.item.InvoiceNumber
	if g.Fake.Chance(0.03) {
		number = strings.TrimLeft(number, "ABCDEFGHIJKLMNOPQRSTUVWXYZ-")
	}
	debtor := it.fi.rec.Customer.Name
	face := facMoney(it.item.FaceMinor)
	switch ev.kind {
	case factoringremittance.RemittanceLineFunding:
		reserve := it.item.FaceMinor - it.item.ExpectedAdvanceMinor - it.item.ExpectedFeeMinor
		return []string{
			number, it.sub.sub.Number, debtor, f.Style.Purchase, face,
			facMoney(it.item.ExpectedAdvanceMinor), facMoney(it.item.ExpectedFeeMinor),
			facMoney(reserve), facMoney(it.item.ExpectedAdvanceMinor), date,
		}
	case factoringremittance.RemittanceLineRejection:
		return []string{
			number, it.sub.sub.Number, debtor, f.Style.Decline + " - " + it.decline, face,
			"", "", "", "", date,
		}
	case factoringremittance.RemittanceLineChargeback:
		repurchase := it.item.ExpectedAdvanceMinor + it.item.ExpectedFeeMinor
		return []string{
			number, it.sub.sub.Number, debtor, f.Style.Chargeback, face,
			"", "", "", facMoney(-repurchase), date,
		}
	default:
		reserve := it.item.FaceMinor - it.item.ExpectedAdvanceMinor - it.item.ExpectedFeeMinor
		extra := min(it.extraFee, reserve)
		fee := ""
		if extra > 0 {
			fee = facMoney(extra)
		}
		return []string{
			number, it.sub.sub.Number, debtor, f.Style.Release, face,
			"", fee, facMoney(reserve), facMoney(reserve - extra), date,
		}
	}
}

func (g *factoringPhase) facCandidates(
	run *facRun,
	f *volumegen.Factor,
) []*factoringremittance.RemittanceCandidate {
	recourse := f.Agreement.Recourse == factoring.RecourseFull
	out := make([]*factoringremittance.RemittanceCandidate, 0)
	for _, fs := range run.subs {
		if fs.factor != f {
			continue
		}
		for _, it := range fs.items {
			if !it.item.Status.IsLive() {
				continue
			}
			out = append(out, &factoringremittance.RemittanceCandidate{
				Item:             it.item,
				SubmissionNumber: fs.sub.Number,
				SentAt:           fs.sub.SentAt,
				Recourse:         recourse,
			})
		}
	}
	return out
}

// facBookLine books one matched line the way remittance apply does: funding
// and rejections through RecordFunding, releases and chargebacks through the
// reserve ledger.
func (g *factoringPhase) facBookLine(
	run *facRun,
	rem *factoringremittance.Remittance,
	line *factoringremittance.RemittanceLine,
	ev *facEvent,
	at int64,
	actor pulid.ID,
) (pulid.ID, error) {
	it := ev.item
	agreement := it.sub.factor.Agreement
	row := "row " + strconv.Itoa(line.RowNumber)
	switch ev.kind {
	case factoringremittance.RemittanceLineFunding:
		if err := it.item.Fund(ev.at, it.item.ExpectedFunding()); err != nil {
			return pulid.Nil, err
		}
		it.item.UpdatedAt = at
		movement, err := factoringledger.PlanFundingMovement(it.item, agreement.AccountingTreatment)
		if err != nil {
			return pulid.Nil, err
		}
		return g.facBook(run, &facBooking{
			item: it, movement: movement, event: accountingcontrol.JournalSourceEventFactoringFunded,
			key: "factoring-funded:" + it.item.ID.String(), on: ev.at, at: at, actor: actor,
			memo: "Factor remittance " + rem.FileName, remitLine: line.ID,
		})
	case factoringremittance.RemittanceLineRejection:
		if err := it.item.Reject("Declined on the factor's remittance " + rem.FileName + ", " + row); err != nil {
			return pulid.Nil, err
		}
		it.item.UpdatedAt = at
		it.fi.rejectedAt = at
		return pulid.Nil, nil
	case factoringremittance.RemittanceLineChargeback:
		movement, err := factoringledger.PlanChargebackMovement(
			it.item,
			agreement.AccountingTreatment,
			factoringledger.ChargebackRepurchaseMinor(it.item),
		)
		if err != nil {
			return pulid.Nil, err
		}
		if err = it.item.ChargeBack(ev.at, "Charged back on the factor's remittance "+rem.FileName+", "+row); err != nil {
			return pulid.Nil, err
		}
		it.item.UpdatedAt = at
		id, err := g.facBook(run, &facBooking{
			item: it, movement: movement, event: accountingcontrol.JournalSourceEventFactoringChargedBack,
			key: "factoring-charged-back:" + it.item.ID.String() + ":1", on: ev.at, at: at,
			actor: actor, memo: it.item.ChargebackReason, remitLine: line.ID,
		})
		it.chargeback = &movement
		it.chargebackID = id
		return id, err
	default:
		_, _, reserve := it.item.Actuals()
		cash := max(reserve-min(it.extraFee, reserve), 0)
		movement, err := factoringledger.PlanReleaseMovement(
			it.item,
			agreement.AccountingTreatment,
			cash,
		)
		if err != nil {
			return pulid.Nil, err
		}
		if err = it.item.Settle(ev.at); err != nil {
			return pulid.Nil, err
		}
		it.item.UpdatedAt = at
		return g.facBook(run, &facBooking{
			item: it, movement: movement, event: accountingcontrol.JournalSourceEventFactoringReserveReleased,
			key: "factoring-reserve-released:" + it.item.ID.String(), on: ev.at, at: at, actor: actor,
			memo: "Released on the factor's remittance " + rem.FileName + ", " + row, remitLine: line.ID,
		})
	}
}

// facReverse undoes a chargeback by hand once the customer paid the factor
// after all, as LedgerService.ReverseChargeback does.
func (g *factoringPhase) facReverse(run *facRun, it *facItem, at int64) error {
	if it.chargeback == nil ||
		it.item.Status != factoringsubmission.SubmissionItemStatusChargedBack {
		it.payOn = 0
		return nil
	}
	movement := factoringledger.ReverseMovement(it.chargeback)
	if err := it.item.ReverseChargeback(); err != nil {
		return err
	}
	it.item.UpdatedAt = at
	_, err := g.facBook(run, &facBooking{
		item: it, movement: movement, event: accountingcontrol.JournalSourceEventFactoringChargebackReversed,
		key: "factoring-chargeback-reversed:" + it.chargebackID.String(), on: at, at: at + 10*volumegen.Minute,
		actor: g.PickUser(
			g.Refs.UserIDs,
		), memo: "The customer paid the factor after the chargeback",
		reverses: it.chargebackID,
	})
	return err
}

type facBooking struct {
	item      *facItem
	movement  factoringledger.Movement
	event     accountingcontrol.JournalSourceEventType
	key       string
	on        int64
	at        int64
	actor     pulid.ID
	memo      string
	remitLine pulid.ID
	reverses  pulid.ID
}

func facDescription(kind factoringledger.ReserveEntryKind, number, schedule string) string {
	switch kind {
	case factoringledger.ReserveEntryFunding:
		return "Factor advance on invoice " + number + ", schedule " + schedule
	case factoringledger.ReserveEntryRelease:
		return "Factor reserve released on invoice " + number + ", schedule " + schedule
	case factoringledger.ReserveEntryChargeback:
		return "Factor chargeback of invoice " + number + ", schedule " + schedule
	case factoringledger.ReserveEntryChargebackReversal:
		return "Factor chargeback reversed on invoice " + number + ", schedule " + schedule
	default:
		return "Factoring of invoice " + number + ", schedule " + schedule
	}
}

// facBook posts a reserve movement: the journal, the append-only reserve
// entry and, for a movement that settles or reopens the invoice, the invoice
// balance and its customer ledger line, as LedgerService.postMovement does.
func (g *factoringPhase) facBook(run *facRun, b *facBooking) (pulid.ID, error) {
	if b.movement.IsEmpty() {
		return pulid.Nil, nil
	}
	it := b.item
	sub := it.sub.sub
	journal, err := g.NewJournal(run.setup, b.on)
	if err != nil {
		return pulid.Nil, err
	}
	accounts := run.accounts[sub.FactoringCompanyID]
	description := facDescription(b.movement.Kind, it.item.InvoiceNumber, sub.Number)
	journal.ActorID = b.actor
	journal.Description = description
	journal.SourceType = facSourceObject
	journal.SourceID = it.item.ID
	journal.Document = it.item.InvoiceNumber
	journal.Event = b.event
	journal.ReferenceType = b.event.String()
	journal.Idempotency = b.key
	journal.Auto = true
	for _, line := range b.movement.Lines {
		journal.Legs = append(journal.Legs, ledgervolume.Leg{
			Account:  accounts.For(line.Role),
			Debit:    line.Debit,
			Credit:   line.Credit,
			Customer: it.item.CustomerID,
		})
	}
	run.rows.Journals = append(run.rows.Journals, journal)

	entry := factoringledger.NewReserveEntry(it.item, sub, &b.movement, b.on)
	row := &factoringledger.ReserveEntry{
		ID:                 volumegen.IDAt("facres_", b.at),
		OrganizationID:     entry.OrganizationID,
		BusinessUnitID:     entry.BusinessUnitID,
		FactoringCompanyID: entry.FactoringCompanyID,
		AgreementID:        entry.AgreementID,
		SubmissionID:       entry.SubmissionID,
		SubmissionItemID:   entry.SubmissionItemID,
		InvoiceID:          entry.InvoiceID,
		CustomerID:         entry.CustomerID,
		Kind:               entry.Kind,
		Treatment:          entry.Treatment,
		CurrencyCode:       entry.CurrencyCode,
		CashMinor:          entry.CashMinor,
		FeeMinor:           entry.FeeMinor,
		ReserveMinor:       entry.ReserveMinor,
		LiabilityMinor:     entry.LiabilityMinor,
		ARMinor:            entry.ARMinor,
		JournalEntryID:     journal.EntryID,
		JournalBatchID:     journal.BatchID,
		AccountingDate:     journal.Date,
		OccurredOn:         b.on,
		ReversesEntryID:    b.reverses,
		RemittanceLineID:   b.remitLine,
		Memo:               b.memo,
		CreatedByID:        b.actor,
		CreatedAt:          b.at,
	}
	if len([]rune(row.Memo)) > factoringledger.MaxReserveMemoLength {
		row.Memo = string([]rune(row.Memo)[:factoringledger.MaxReserveMemoLength])
	}
	if row.CurrencyCode == "" {
		row.CurrencyCode = money.DefaultCurrencyCode
	}
	run.entries = append(run.entries, row)

	inv := it.fi.inv
	signed := int64(0)
	switch b.movement.InvoiceEffect {
	case factoringledger.InvoiceEffectSettle:
		inv.ApplyPaymentMinor(b.movement.InvoiceMinor)
		signed = -b.movement.InvoiceMinor
	case factoringledger.InvoiceEffectReopen:
		inv.RemovePaymentMinor(b.movement.InvoiceMinor)
		signed = b.movement.InvoiceMinor
	case factoringledger.InvoiceEffectNone:
	}
	if signed != 0 {
		it.fi.touched = true
		it.fi.updatedAt = max(it.fi.updatedAt, b.at)
		run.rows.Ledger = append(run.rows.Ledger, &customerledger.CustomerLedgerEntry{
			ID:               volumegen.IDAt("cledg_", b.at),
			OrganizationID:   inv.OrganizationID,
			BusinessUnitID:   inv.BusinessUnitID,
			CustomerID:       inv.CustomerID,
			SourceObjectType: facSourceObject,
			SourceObjectID:   row.ID.String(),
			SourceEventType:  b.event.String(),
			RelatedInvoiceID: inv.ID,
			DocumentNumber:   sub.Number,
			TransactionDate:  journal.Date,
			LineNumber:       1,
			AmountMinor:      signed,
			CreatedByID:      b.actor,
			CreatedAt:        b.at,
		})
	}
	return row.ID, nil
}

// facExclude takes declined invoices out of factoring, as a biller does from
// the factoring page, and records the customer paying the organization
// directly once it does.
func (g *factoringPhase) facExclude(run *facRun) error {
	now := volumegen.Unix(g.Now)
	byCustomer := make(map[pulid.ID]*volumegen.CustomerRecord, len(g.Customers))
	for _, c := range g.Customers {
		byCustomer[c.ID] = c
	}
	for _, fi := range run.invoices {
		if fi.rejectedAt == 0 {
			continue
		}
		at := fi.rejectedAt + g.Fake.Between64(4*volumegen.Hour, 3*volumegen.Day)
		if at > now {
			continue
		}
		number := ""
		for _, it := range fi.items {
			if it.item.Status == factoringsubmission.SubmissionItemStatusRejected {
				number = it.sub.sub.Number
			}
		}
		fi.inv.FactoringOverride = invoice.FactoringOverrideExclude
		fi.inv.FactoringOverrideReason = "Declined by the factor on schedule " + number +
			"; collected directly"
		fi.inv.FactoringCompanyID = nil
		fi.inv.FactoringAgreementID = nil
		fi.touched = true
		fi.updatedAt = max(fi.updatedAt, at)
		run.excluded++

		open := fi.inv.OpenBalanceMinor()
		if !run.setup.PostPayments || open <= 0 {
			continue
		}
		due := fi.inv.InvoiceDate + 30*volumegen.Day
		if fi.inv.DueDate != nil {
			due = *fi.inv.DueDate
		}
		paid := max(due+g.Fake.Between64(-5, 25)*volumegen.Day, at+2*volumegen.Day)
		if paid > now-volumegen.Hour || !g.Fake.Chance(0.85) {
			continue
		}
		cus := byCustomer[fi.inv.CustomerID]
		if cus == nil {
			continue
		}
		if err := g.Payment(run.setup, run.rows, cus, paid, []*ledgervolume.PaymentPlan{{
			Invoice: fi.inv, Amount: open, Date: paid,
		}}); err != nil {
			return err
		}
		fi.updatedAt = max(fi.updatedAt, paid)
	}
	return nil
}

// facStates writes each invoice's readiness as the factoring watch last left
// it, using the domain's own evaluation.
func (g *factoringPhase) facStates(run *facRun) {
	now := volumegen.Unix(g.Now)
	for _, fi := range run.invoices {
		inv := fi.inv
		evaluated := now - g.Fake.Between64(volumegen.Minute, 50*volumegen.Minute)
		facts := factoringreadiness.InvoiceFacts{
			Posted:           inv.Status == invoice.StatusPosted,
			Voided:           inv.Status == invoice.StatusVoided,
			Excluded:         inv.FactoringOverride.Excludes(),
			OpenBalanceMinor: inv.OpenBalanceMinor(),
		}
		if inv.FactoringCompanyID != nil {
			facts.FactoringCompanyID = *inv.FactoringCompanyID
		}
		in := &factoringreadiness.ReadinessInputs{At: evaluated, Invoice: facts}
		if f := fi.factor; f != nil && inv.FactoringCompanyID != nil {
			if f.Agreement.InForce(evaluated) {
				in.Agreement = f.Agreement
			}
			if noa := f.Noas[inv.CustomerID]; noa != nil &&
				noa.Status != factoring.NoaStatusReleased {
				in.Noa = noa
			}
			in.OpenDispute = inv.DisputeStatus == invoice.DisputeStatusDisputed
			in.Shipments = []factoringreadiness.ShipmentFacts{{
				ShipmentID: fi.rec.ID,
				ProNumber:  fi.rec.ProNumber,
				Available:  true,
			}}
			for _, it := range fi.items {
				switch {
				case it.item.Status.IsLive():
					in.SubmissionNumber = it.sub.sub.Number
					in.SubmissionItemStatus = it.item.Status
				case it.item.Status == factoringsubmission.SubmissionItemStatusChargedBack:
					in.ChargedBack = true
				}
			}
		}
		readiness := factoringreadiness.EvaluateReadiness(in)
		posted := inv.InvoiceDate
		if inv.PostedAt != nil {
			posted = *inv.PostedAt
		}
		state := &factoringreadiness.InvoiceState{
			InvoiceID:          inv.ID,
			OrganizationID:     inv.OrganizationID,
			BusinessUnitID:     inv.BusinessUnitID,
			FactoringCompanyID: inv.FactoringCompanyID,
			CustomerID:         inv.CustomerID,
			Verdict:            readiness.Verdict,
			Checks:             readiness.Checks,
			FailedChecks:       readiness.FailedCount(),
			OpenBalanceMinor:   inv.OpenBalanceMinor(),
			EvaluatedAt:        evaluated,
			CreatedAt:          min(posted+volumegen.Minute, evaluated),
		}
		if readiness.Verdict == factoringreadiness.VerdictReady {
			since := fi.readyAt
			for _, it := range fi.items {
				if it.item.ReleasedAt != nil {
					since = max(since, *it.item.ReleasedAt)
				}
			}
			if since == 0 || since > evaluated {
				since = min(posted+volumegen.Minute, evaluated)
			}
			state.ReadySince = volumegen.Ptr(since)
			state.ReadyAnnouncedAt = volumegen.Ptr(
				min(since+g.Fake.Between64(volumegen.Minute, 10*volumegen.Minute), evaluated),
			)
		}
		run.states = append(run.states, state)
	}
}

// facOddRemittances adds the files that never book anything: one whose
// columns need mapping, one that could not be read and a duplicate that was
// discarded.
func (g *factoringPhase) facOddRemittances(run *facRun, now int64) {
	if len(run.remittances) == 0 {
		return
	}
	f := run.subs[0].factor
	uploader := g.PickUser(g.Refs.UserIDs)

	mappingAt := now - g.Fake.Between64(2*volumegen.Hour, 2*volumegen.Day)
	headers := []string{"Client", "Doc #", "Purch Dt", "Gross", "Holdback", "Net Funds"}
	mapping := factoringremit.GuessMapping(headers)
	needs := &factoringremittance.Remittance{
		ID:                 volumegen.IDAt("facrem_", mappingAt),
		OrganizationID:     g.Refs.OrgID,
		BusinessUnitID:     g.Refs.BuID,
		FactoringCompanyID: f.Company.ID,
		FileName:           strings.ToUpper(f.Company.Code) + "_month_end_statement.csv",
		Source:             factoringremittance.RemittanceSourceUpload,
		SourceFormat:       factoringremittance.RemittanceFormatCsv,
		Status:             factoringremittance.RemittanceStatusNeedsMapping,
		DefaultKind:        factoringremittance.RemittanceLineReserveRelease,
		CurrencyCode:       money.DefaultCurrencyCode,
		Mapping:            mapping.Strings(),
		Headers:            headers,
		UnmappedHeaders:    factoringremit.UnmappedHeaders(headers, mapping),
		StagedAt:           volumegen.Ptr(mappingAt + volumegen.Minute),
		CreatedByID:        uploader,
	}
	needsDoc := g.DocumentRow(
		factoringremittance.RemittanceDocumentResourceType,
		needs.ID,
		needs.FileName,
		"text/csv",
		document.ProcessingProfileNone,
		mappingAt,
		uploader,
		18_400,
	)
	needs.DocumentID = needsDoc.ID

	failedAt := now - g.Fake.Between64(3*volumegen.Day, 20*volumegen.Day)
	failed := &factoringremittance.Remittance{
		ID:                 volumegen.IDAt("facrem_", failedAt),
		OrganizationID:     g.Refs.OrgID,
		BusinessUnitID:     g.Refs.BuID,
		FactoringCompanyID: f.Company.ID,
		FileName:           strings.ToUpper(f.Company.Code) + "_funding_report_scan.pdf",
		Source:             factoringremittance.RemittanceSourceUpload,
		SourceFormat:       factoringremittance.RemittanceFormatPdf,
		Status:             factoringremittance.RemittanceStatusFailed,
		DefaultKind:        factoringremittance.RemittanceLineFunding,
		CurrencyCode:       money.DefaultCurrencyCode,
		Error:              "The PDF is a scanned image with no readable table of invoices",
		StagedAt:           volumegen.Ptr(failedAt + 3*volumegen.Minute),
		CreatedByID:        uploader,
	}
	failedDoc := g.DocumentRow(
		factoringremittance.RemittanceDocumentResourceType,
		failed.ID,
		failed.FileName,
		"application/pdf",
		document.ProcessingProfileFactoringRemittance,
		failedAt,
		uploader,
		812_000,
	)
	failed.DocumentID = failedDoc.ID

	original := run.remittances[g.Fake.Intn(len(run.remittances))]
	dupAt := (*original.StagedAt) + g.Fake.Between64(volumegen.Hour, 6*volumegen.Hour)
	discarded := &factoringremittance.Remittance{
		ID:                 volumegen.IDAt("facrem_", dupAt),
		OrganizationID:     g.Refs.OrgID,
		BusinessUnitID:     g.Refs.BuID,
		FactoringCompanyID: original.FactoringCompanyID,
		FileName:           original.FileName,
		Source:             factoringremittance.RemittanceSourceUpload,
		SourceFormat:       original.SourceFormat,
		Status:             factoringremittance.RemittanceStatusDiscarded,
		DefaultKind:        factoringremittance.RemittanceLineFunding,
		CurrencyCode:       money.DefaultCurrencyCode,
		DiscardedAt:        volumegen.Ptr(dupAt + 20*volumegen.Minute),
		DiscardReason:      "Duplicate upload of " + original.FileName,
		CreatedByID:        uploader,
	}

	run.remittances = append(run.remittances, needs, failed, discarded)
	run.docs = append(run.docs, needsDoc, failedDoc)
	run.remStamps = append(run.remStamps,
		volumegen.Stamp{Id: needs.ID, Created: mappingAt, Updated: mappingAt + volumegen.Minute},
		volumegen.Stamp{Id: failed.ID, Created: failedAt, Updated: failedAt + 3*volumegen.Minute},
		volumegen.Stamp{Id: discarded.ID, Created: dupAt, Updated: dupAt + 20*volumegen.Minute},
	)
}

//nolint:gocognit,funlen // writes follow the foreign keys in one place
func (g *factoringPhase) facWrite(ctx context.Context, tx *keel.DB, run *facRun) error {
	if len(run.docs) > 0 {
		marks := make([]volumegen.Stamp, 0, len(run.docs))
		for _, d := range run.docs {
			marks = append(
				marks,
				volumegen.Stamp{Id: d.ID, Created: d.CreatedAt, Updated: d.CreatedAt},
			)
		}
		if err := volumegen.InsertRows(ctx, tx, documentdb.Documents.Handle, run.docs); err != nil {
			return fmt.Errorf("insert remittance documents: %w", err)
		}
		if err := volumegen.Backdate(ctx, tx, documentdb.Documents.Handle, marks); err != nil {
			return err
		}
	}

	subs := make([]*factoringsubmission.Submission, 0, len(run.subs))
	items := make([]*factoringsubmission.SubmissionItem, 0)
	subMarks := make([]volumegen.Stamp, 0, len(run.subs))
	itemMarks := make([]volumegen.Stamp, 0)
	for _, fs := range run.subs {
		subs = append(subs, fs.sub)
		subMarks = append(
			subMarks,
			volumegen.Stamp{Id: fs.sub.ID, Created: fs.slot, Updated: fs.sub.UpdatedAt},
		)
		for _, it := range fs.items {
			items = append(items, it.item)
			itemMarks = append(
				itemMarks,
				volumegen.Stamp{Id: it.item.ID, Created: fs.slot, Updated: it.item.UpdatedAt},
			)
		}
	}
	if err := volumegen.InsertRows(ctx, tx, factoringsubmissiondb.Submissions.Handle, subs); err != nil {
		return fmt.Errorf("insert factoring submissions: %w", err)
	}
	if err := volumegen.InsertRows(ctx, tx, factoringsubmissiondb.SubmissionItems.Handle, items); err != nil {
		return fmt.Errorf("insert factoring submission items: %w", err)
	}

	pending := make(map[pulid.ID]pulid.ID, len(run.applied))
	for _, a := range run.applied {
		pending[a.line.ID] = a.entryID
		a.line.MatchStatus = factoringremittance.RemittanceLineMatched
		a.line.ReserveEntryID = pulid.Nil
	}
	if err := volumegen.InsertRows(ctx, tx, factoringremittancedb.Remittances.Handle, run.remittances); err != nil {
		return fmt.Errorf("insert factoring remittances: %w", err)
	}
	if err := volumegen.InsertRows(ctx, tx, factoringremittancedb.RemittanceLines.Handle, run.lines); err != nil {
		return fmt.Errorf("insert factoring remittance lines: %w", err)
	}
	profileMarks := make([]volumegen.Stamp, 0, len(run.profiles))
	for _, p := range run.profiles {
		profileMarks = append(
			profileMarks,
			volumegen.Stamp{Id: p.ID, Created: p.CreatedAt, Updated: p.CreatedAt},
		)
	}
	if err := volumegen.InsertRows(ctx, tx, factoringremittancedb.RemittanceProfiles.Handle, run.profiles); err != nil {
		return fmt.Errorf("insert factoring remittance profiles: %w", err)
	}

	if err := g.Write(ctx, tx, run.rows); err != nil {
		return err
	}
	if err := volumegen.InsertRows(ctx, tx, factoringledgerdb.ReserveEntries.Handle, run.entries); err != nil {
		return fmt.Errorf("insert factoring reserve entries: %w", err)
	}

	if len(pending) > 0 {
		lineIDs := make([]string, 0, len(pending))
		entryIDs := make([]string, 0, len(pending))
		for lineID, entryID := range pending {
			lineIDs = append(lineIDs, lineID.String())
			entryIDs = append(entryIDs, entryID.String())
		}
		rl := factoringremittancedb.RemittanceLines
		d := keel.Unnest("d", []string{"id", "reserve_entry_id"}, lineIDs, entryIDs)
		if _, err := rl.With(d).Join(d, d.Col("id").EqCol(rl.ID)).Where(keel.True()).Update(
			ctx,
			tx,
			rl.MatchStatus.To(factoringremittance.RemittanceLineApplied),
			rl.ReserveEntryID.ToExpr(d.Col("reserve_entry_id")),
			volumegen.KeepUpdated(rl.Handle),
		); err != nil {
			return fmt.Errorf("mark remittance lines applied: %w", err)
		}
	}

	if err := g.facUpdateInvoices(ctx, tx, run); err != nil {
		return err
	}
	if err := volumegen.InsertRows(ctx, tx, factoringreadinessdb.InvoiceStates.Handle, run.states); err != nil {
		return fmt.Errorf("insert factoring invoice states: %w", err)
	}

	for _, b := range []struct {
		table string
		marks []volumegen.Stamp
	}{
		{"factoring_submissions", subMarks},
		{"factoring_submission_items", itemMarks},
		{"factoring_remittances", run.remStamps},
		{"factoring_remittance_lines", run.lineStamps},
	} {
		if err := volumegen.BackdateTable(ctx, tx, b.table, b.marks); err != nil {
			return err
		}
	}
	if len(profileMarks) > 0 {
		if err := volumegen.Backdate(ctx, tx, factoringremittancedb.RemittanceProfiles.Handle, profileMarks); err != nil {
			return err
		}
	}

	return nil
}

func (g *factoringPhase) facUpdateInvoices(ctx context.Context, tx *keel.DB, run *facRun) error {
	var ids, settlements, overrides, reasons, factors, agreements []string
	var applied []string
	var appliedMinor, updated []int64
	for _, fi := range run.invoices {
		if !fi.touched {
			continue
		}
		inv := fi.inv
		ids = append(ids, inv.ID.String())
		applied = append(applied, inv.AppliedAmount.StringFixed(4))
		appliedMinor = append(appliedMinor, inv.AppliedAmountMinor)
		settlements = append(settlements, string(inv.SettlementStatus))
		overrides = append(overrides, string(inv.FactoringOverride))
		reasons = append(reasons, inv.FactoringOverrideReason)
		factor, agreement := "", ""
		if inv.FactoringCompanyID != nil {
			factor = inv.FactoringCompanyID.String()
		}
		if inv.FactoringAgreementID != nil {
			agreement = inv.FactoringAgreementID.String()
		}
		factors = append(factors, factor)
		agreements = append(agreements, agreement)
		updated = append(updated, max(fi.updatedAt, inv.UpdatedAt))
		inv.UpdatedAt = max(fi.updatedAt, inv.UpdatedAt)
	}
	for begin := 0; begin < len(ids); begin += volumegen.BackdateChunk {
		end := min(begin+volumegen.BackdateChunk, len(ids))
		d := keel.Unnest("d", []string{
			"id", "applied_amount", "applied_amount_minor", "settlement_status", "factoring_override",
			"factoring_override_reason", "factoring_company_id", "factoring_agreement_id", "updated_at",
		}, ids[begin:end], applied[begin:end], appliedMinor[begin:end], settlements[begin:end],
			overrides[begin:end], reasons[begin:end], factors[begin:end], agreements[begin:end],
			updated[begin:end])
		inv := invoicedb.Invoices
		if _, err := inv.With(d).Join(d, d.Col("id").EqCol(inv.ID)).Where(keel.True()).Update(
			ctx,
			tx,
			inv.AppliedAmount.ToExpr(keel.CastNumeric(d.Col("applied_amount"))),
			inv.AppliedAmountMinor.ToExpr(d.Col("applied_amount_minor")),
			inv.SettlementStatus.ToExpr(d.Col("settlement_status")),
			inv.FactoringOverride.ToExpr(d.Col("factoring_override")),
			inv.FactoringOverrideReason.ToExpr(volumegen.NullIfEmpty(d.Col("factoring_override_reason"))),
			inv.FactoringCompanyID.ToExpr(volumegen.NullIfEmpty(d.Col("factoring_company_id"))),
			inv.FactoringAgreementID.ToExpr(volumegen.NullIfEmpty(d.Col("factoring_agreement_id"))),
			inv.UpdatedAt.ToExpr(d.Col("updated_at")),
		); err != nil {
			return fmt.Errorf("update factored invoices: %w", err)
		}
	}
	return nil
}

func (g *factoringPhase) facCount(run *facRun) {
	items, funded, rejected, settled, chargedBack := 0, 0, 0, 0, 0
	for _, fs := range run.subs {
		for _, it := range fs.items {
			items++
			switch it.item.Status {
			case factoringsubmission.SubmissionItemStatusFunded:
				funded++
			case factoringsubmission.SubmissionItemStatusRejected:
				rejected++
			case factoringsubmission.SubmissionItemStatusSettled:
				settled++
			case factoringsubmission.SubmissionItemStatusChargedBack:
				chargedBack++
			case factoringsubmission.SubmissionItemStatusPending,
				factoringsubmission.SubmissionItemStatusReleased:
			}
		}
	}
	verdicts := make(map[factoringreadiness.Verdict]int)
	for _, s := range run.states {
		verdicts[s.Verdict]++
	}
	g.Report.Add("Factoring schedules", len(run.subs))
	g.Report.Add("Factoring schedule items", items)
	g.Report.Add("Factoring reserve entries", len(run.entries))
	g.Report.Add("Factor remittances", len(run.remittances))
	g.Report.Add("Factor remittance lines", len(run.lines))
	g.Report.Add("Factoring invoice states", len(run.states))
	g.Report.Add("Factoring payments collected directly", len(run.rows.Payments))
	g.Report.Add("Customer ledger entries", len(run.rows.Ledger))
	g.Report.Add("Journal entries", len(run.rows.Entries))
	g.Opts.Logf(
		"  • factoring items: %d funded, %d settled, %d charged back, %d declined (%d excluded)",
		funded,
		settled,
		chargedBack,
		rejected,
		run.excluded,
	)
	g.Opts.Logf(
		"  • factoring readiness: %d ready, %d not ready, %d submitted, %d funded, %d excluded",
		verdicts[factoringreadiness.VerdictReady],
		verdicts[factoringreadiness.VerdictNotReady],
		verdicts[factoringreadiness.VerdictSubmitted],
		verdicts[factoringreadiness.VerdictFunded],
		verdicts[factoringreadiness.VerdictExcluded],
	)
}

const (
	facSourceObject = "FactoringSubmissionItem"
)

var facDeclineReasons = []string{
	"Debtor over credit limit",
	"Debtor not approved",
	"Duplicate of an invoice already purchased",
	"Proof of delivery illegible",
	"Invoice older than purchase terms",
}

var facFailures = []string{
	"550 5.7.1 Message rejected: attachment size exceeds the recipient's limit",
	"421 4.7.0 Temporary server error at the factor's mail gateway; retries exhausted",
	"Connection to the factor's mail server timed out",
}

type factoringPhase struct {
	*ledgervolume.Ledger
}

func Seed(ctx context.Context, g *volumegen.Generator) error {
	return (&factoringPhase{Ledger: &ledgervolume.Ledger{Generator: g}}).seedFactoring(ctx)
}
