package watchtowerfactoring

import (
	"context"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/watchtoweritem"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringremittanceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringwatchtowerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	factoringReadyPrefix        = "ready:"
	factoringMissingNoaPrefix   = "noa-missing:"
	factoringExpiringNoaPrefix  = "noa-expiring:"
	factoringSendFailedPrefix   = "send-failed:"
	factoringUnfundedPrefix     = "unfunded:"
	factoringChargebackPrefix   = "chargeback-due:"
	factoringRemittancePrefix   = "remittance-exceptions:"
	pathFactoringQueue          = "/billing/factoring"
	FactoringReadyCriticalAfter = int64(3 * timeutils.SecondsPerDay)
	FactoringNoaCriticalWithin  = int64(7 * timeutils.SecondsPerDay)
	pathFactoringInvoices       = "/billing/invoices"
	pathFactoringCustomers      = "/billing/configuration-files/customers"
	factoringDateLayout         = "January 2, 2006"
	maxFactoringTitleLength     = 200
)

type FactoringSource struct {
	states      factoringports.InvoiceStateRepository
	reader      factoringwatchtowerports.WatchtowerReader
	remittances factoringremittanceports.RemittanceRepository
	now         func() int64
}

func NewFactoringSource(
	states factoringports.InvoiceStateRepository,
	reader factoringwatchtowerports.WatchtowerReader,
	remittances factoringremittanceports.RemittanceRepository,
) watchtowerports.WatchtowerSource {
	return &FactoringSource{
		states:      states,
		reader:      reader,
		remittances: remittances,
		now:         timeutils.NowUnix,
	}
}

func (s *FactoringSource) Kind() watchtower.SourceKind { return watchtower.SourceFactoring }

type factoringSnapshot struct {
	ready       []*factoringreadiness.InvoiceState
	labels      map[pulid.ID]*factoringwatchtowerports.InvoiceLabel
	missing     []*factoringwatchtowerports.CustomerMissingNoa
	expiring    []*factoringwatchtowerports.ExpiringNoa
	failed      []*factoringwatchtowerports.FailedSubmission
	unfunded    []*factoringwatchtowerports.UnfundedSubmission
	chargebacks []*factoringwatchtowerports.ChargebackDue
	remittances []*factoringremittanceports.OpenRemittance
}

func (s *FactoringSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	now := s.now()
	snap, err := s.read(ctx, tenant, now)
	if err != nil {
		return nil, err
	}
	return snap.describe(tenant, now), nil
}

func (s *FactoringSource) read(
	ctx context.Context,
	tenant pagination.TenantInfo,
	now int64,
) (*factoringSnapshot, error) {
	snap := &factoringSnapshot{}
	var err error
	snap.ready, err = s.states.ListReadyUnsubmitted(ctx, factoringports.ListReadyUnsubmittedRequest{
		TenantInfo:  tenant,
		ReadyBefore: now - factoringreadiness.ReadyUnsubmittedSeconds,
		Limit:       watchtoweritem.SnapshotLimit,
	})
	if err != nil {
		return nil, err
	}

	invoiceIDs := make([]pulid.ID, 0, len(snap.ready))
	for _, state := range snap.ready {
		if state != nil {
			invoiceIDs = append(invoiceIDs, state.InvoiceID)
		}
	}
	if snap.labels, err = s.reader.ListInvoiceLabels(ctx, &factoringwatchtowerports.ListInvoiceLabelsRequest{
		TenantInfo: tenant,
		InvoiceIDs: invoiceIDs,
	}); err != nil {
		return nil, err
	}
	if snap.missing, err = s.reader.ListCustomersMissingNoa(
		ctx,
		&factoringwatchtowerports.ListCustomersMissingNoaRequest{
			TenantInfo: tenant,
			At:         now,
			Limit:      watchtoweritem.SnapshotLimit,
		},
	); err != nil {
		return nil, err
	}
	if snap.expiring, err = s.reader.ListNoasExpiringSoon(
		ctx,
		&factoringwatchtowerports.ListNoasExpiringSoonRequest{
			TenantInfo: tenant,
			At:         now,
			Before:     now + FactoringNoaWarningSeconds(),
			Limit:      watchtoweritem.SnapshotLimit,
		},
	); err != nil {
		return nil, err
	}
	if snap.failed, err = s.reader.ListFailedSubmissions(
		ctx,
		&factoringwatchtowerports.ListFailedSubmissionsRequest{TenantInfo: tenant, Limit: watchtoweritem.SnapshotLimit},
	); err != nil {
		return nil, err
	}
	if snap.unfunded, err = s.reader.ListUnfundedSubmissions(
		ctx,
		&factoringwatchtowerports.ListUnfundedSubmissionsRequest{
			TenantInfo: tenant,
			At:         now,
			Limit:      watchtoweritem.SnapshotLimit,
		},
	); err != nil {
		return nil, err
	}
	if snap.chargebacks, err = s.reader.ListChargebacksDue(
		ctx,
		&factoringwatchtowerports.ListChargebacksDueRequest{
			TenantInfo: tenant,
			Before:     now + FactoringChargebackWarningSeconds(),
			Limit:      watchtoweritem.SnapshotLimit,
		},
	); err != nil {
		return nil, err
	}
	if snap.remittances, err = s.remittances.ListOpen(
		ctx,
		&factoringremittanceports.ListOpenRemittancesRequest{TenantInfo: tenant, Limit: watchtoweritem.SnapshotLimit},
	); err != nil {
		return nil, err
	}
	return snap, nil
}

func (snap *factoringSnapshot) describe(
	tenant pagination.TenantInfo,
	now int64,
) []watchtowerports.WatchtowerItemInput {
	items := make(
		[]watchtowerports.WatchtowerItemInput,
		0,
		len(snap.ready)+len(snap.missing)+len(snap.expiring)+len(snap.failed)+
			len(snap.unfunded)+len(snap.chargebacks)+len(snap.remittances),
	)
	keep := func(item watchtowerports.WatchtowerItemInput, open bool) {
		if open {
			items = append(items, item)
		}
	}
	for _, state := range snap.ready {
		if state != nil {
			keep(DescribeFactoringReadyUnsubmitted(
				tenant,
				state,
				snap.labels[state.InvoiceID],
				now,
			))
		}
	}
	for _, row := range snap.missing {
		keep(DescribeFactoringMissingNoa(tenant, row, now))
	}
	for _, row := range snap.expiring {
		keep(DescribeFactoringNoaExpiring(tenant, row, now))
	}
	for _, row := range snap.failed {
		keep(DescribeFactoringSendFailed(tenant, row))
	}
	for _, row := range snap.unfunded {
		keep(DescribeFactoringUnfunded(tenant, row, now))
	}
	for _, row := range snap.chargebacks {
		keep(DescribeFactoringChargebackDue(tenant, row, now))
	}
	for _, row := range snap.remittances {
		keep(DescribeFactoringRemittanceExceptions(tenant, row))
	}
	return items
}

func FactoringChargebackWarningSeconds() int64 {
	return int64(factoring.ChargebackWarningDays) * timeutils.SecondsPerDay
}

func FactoringUnfundedSourceID(submissionID pulid.ID) string {
	return factoringUnfundedPrefix + submissionID.String()
}

func FactoringChargebackDueSourceID(itemID pulid.ID) string {
	return factoringChargebackPrefix + itemID.String()
}

func DescribeFactoringUnfunded(
	tenant pagination.TenantInfo,
	row *factoringwatchtowerports.UnfundedSubmission,
	now int64,
) (watchtowerports.WatchtowerItemInput, bool) {
	if row == nil || row.SubmissionID.IsNil() || row.PendingCount <= 0 || row.DueAt >= now {
		return watchtowerports.WatchtowerItemInput{}, false
	}

	factorName := stringutils.FirstNonEmpty(row.FactorName, "the factor")
	title := "Schedule " + row.Number + " has not been funded by " + factorName
	summary := strconv.Itoa(row.PendingCount) + " invoices are still waiting on " +
		money.FormatMinor(row.ExpectedAdvanceMinor, row.CurrencyCode) +
		" in advances, due by " + factoringDate(row.DueAt) +
		". Ask the factor about the schedule, then record its funding or rejections."

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourceFactoring,
		SourceID:    FactoringUnfundedSourceID(row.SubmissionID),
		Severity:    watchtower.SeverityWarning,
		Title:       stringutils.TruncateRunes(title, maxFactoringTitleLength),
		Summary:     stringutils.OneLine(summary, watchtoweritem.MaxSummaryLength),
		SubjectType: agent.SubjectFactoringSubmission,
		SubjectID:   row.SubmissionID,
		Path: factoringRecordPath(
			"factoring_submission",
			row.SubmissionID,
			pathFactoringQueue,
		),
		OccurredAt: row.DueAt,
	}, true
}

func DescribeFactoringChargebackDue(
	tenant pagination.TenantInfo,
	row *factoringwatchtowerports.ChargebackDue,
	now int64,
) (watchtowerports.WatchtowerItemInput, bool) {
	if row == nil || row.ItemID.IsNil() || row.InvoiceID.IsNil() ||
		row.DeadlineAt-now > FactoringChargebackWarningSeconds() {
		return watchtowerports.WatchtowerItemInput{}, false
	}

	factorName := stringutils.FirstNonEmpty(row.FactorName, "The factor")
	invoiceName := "Invoice " + stringutils.FirstNonEmpty(row.InvoiceNumber, row.InvoiceID.String())
	title := invoiceName + " can be charged back " + remainingLabel(max(row.DeadlineAt-now, 0))
	if row.DeadlineAt <= now {
		title = invoiceName + " is past its chargeback window"
	}
	summary := factorName + " funded it on " + factoringDate(row.FundedOn)
	if row.CustomerName != "" {
		summary += " and " + row.CustomerName + " has not paid it"
	}
	summary += ". From " + factoringDate(row.DeadlineAt) + " the factor can charge it back, and " +
		"the organization would owe " + money.FormatMinor(row.RepurchaseMinor, row.CurrencyCode) +
		". Chase the customer's payment now."

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourceFactoring,
		SourceID:    FactoringChargebackDueSourceID(row.ItemID),
		Severity:    watchtower.SeverityCritical,
		Title:       stringutils.TruncateRunes(title, maxFactoringTitleLength),
		Summary:     stringutils.OneLine(summary, watchtoweritem.MaxSummaryLength),
		SubjectType: agent.SubjectInvoice,
		SubjectID:   row.InvoiceID,
		EventKind:   agenttrigger.EventFactoringChargebackDue,
		Path:        factoringRecordPath("invoice", row.InvoiceID, pathFactoringInvoices),
		OccurredAt:  max(row.DeadlineAt-FactoringChargebackWarningSeconds(), row.FundedOn, 1),
	}, true
}

func FactoringNoaWarningSeconds() int64 {
	return int64(factoring.NoaExpiryWarningDays) * timeutils.SecondsPerDay
}

func FactoringReadySourceID(invoiceID pulid.ID) string {
	return factoringReadyPrefix + invoiceID.String()
}

func FactoringMissingNoaSourceID(customerID, factoringCompanyID pulid.ID) string {
	return factoringMissingNoaPrefix + customerID.String() + ":" + factoringCompanyID.String()
}

func FactoringExpiringNoaSourceID(noaID pulid.ID) string {
	return factoringExpiringNoaPrefix + noaID.String()
}

func FactoringSendFailedSourceID(submissionID pulid.ID) string {
	return factoringSendFailedPrefix + submissionID.String()
}

func DescribeFactoringSendFailed(
	tenant pagination.TenantInfo,
	row *factoringwatchtowerports.FailedSubmission,
) (watchtowerports.WatchtowerItemInput, bool) {
	if row == nil || row.SubmissionID.IsNil() || row.FailedAt <= 0 {
		return watchtowerports.WatchtowerItemInput{}, false
	}

	factorName := stringutils.FirstNonEmpty(row.FactorName, "the factor")
	title := "Schedule " + row.Number + " could not be sent to " + factorName
	summary := strconv.Itoa(row.InvoiceCount) + " invoices (" +
		money.FormatMinor(row.FaceMinor, row.CurrencyCode) + ") did not reach " +
		stringutils.FirstNonEmpty(row.TransportTarget, factorName) + " after every retry"
	if row.LastError != "" {
		summary += ": " + row.LastError
	}
	summary += ". Fix the factor's delivery settings and retry, or cancel the schedule to " +
		"release its invoices."

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourceFactoring,
		SourceID:    FactoringSendFailedSourceID(row.SubmissionID),
		Severity:    watchtower.SeverityCritical,
		Title:       stringutils.TruncateRunes(title, maxFactoringTitleLength),
		Summary:     stringutils.OneLine(summary, watchtoweritem.MaxSummaryLength),
		SubjectType: agent.SubjectFactoringSubmission,
		SubjectID:   row.SubmissionID,
		EventKind:   agenttrigger.EventFactoringSubmissionFailed,
		Path: factoringRecordPath(
			"factoring_submission",
			row.SubmissionID,
			pathFactoringQueue,
		),
		OccurredAt: row.FailedAt,
	}, true
}

func DescribeFactoringReadyUnsubmitted(
	tenant pagination.TenantInfo,
	state *factoringreadiness.InvoiceState,
	label *factoringwatchtowerports.InvoiceLabel,
	now int64,
) (watchtowerports.WatchtowerItemInput, bool) {
	if state == nil || state.InvoiceID.IsNil() || state.ReadySince == nil ||
		!state.ReadyLongerThan(now, factoringreadiness.ReadyUnsubmittedSeconds) {
		return watchtowerports.WatchtowerItemInput{}, false
	}

	waited := now - *state.ReadySince
	severity := watchtower.SeverityWarning
	if waited >= FactoringReadyCriticalAfter {
		severity = watchtower.SeverityCritical
	}

	invoiceName := "An invoice"
	currency := ""
	customerName := ""
	factorName := ""
	if label != nil {
		if label.Number != "" {
			invoiceName = "Invoice " + label.Number
		}
		currency = label.CurrencyCode
		customerName = label.CustomerName
		factorName = label.FactorName
	}

	title := invoiceName + " has been ready to factor for " + waitedLabel(waited)
	summary := money.FormatMinor(state.OpenBalanceMinor, currency) + " is open"
	if customerName != "" {
		summary += " for " + customerName
	}
	summary += " and every check has passed. Submit it to " +
		stringutils.FirstNonEmpty(factorName, "the factor") + " so it funds."

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourceFactoring,
		SourceID:    FactoringReadySourceID(state.InvoiceID),
		Severity:    severity,
		Title:       stringutils.TruncateRunes(title, maxFactoringTitleLength),
		Summary:     stringutils.OneLine(summary, watchtoweritem.MaxSummaryLength),
		SubjectType: agent.SubjectInvoice,
		SubjectID:   state.InvoiceID,
		EventKind:   agenttrigger.EventFactoringInvoiceReady,
		Path:        factoringRecordPath("invoice", state.InvoiceID, pathFactoringInvoices),
		OccurredAt:  *state.ReadySince + factoringreadiness.ReadyUnsubmittedSeconds,
	}, true
}

func DescribeFactoringMissingNoa(
	tenant pagination.TenantInfo,
	row *factoringwatchtowerports.CustomerMissingNoa,
	now int64,
) (watchtowerports.WatchtowerItemInput, bool) {
	if row == nil || row.CustomerID.IsNil() || row.FactoringCompanyID.IsNil() {
		return watchtowerports.WatchtowerItemInput{}, false
	}

	var subjectType agent.SubjectType
	var subjectID pulid.ID
	hasNotice := row.NoaID.IsNotNil()
	if hasNotice {
		if row.NoaStatus == factoring.NoaStatusAcknowledged &&
			(row.NoaExpiresAt == nil || *row.NoaExpiresAt > now) {
			return watchtowerports.WatchtowerItemInput{}, false
		}
		subjectType = agent.SubjectFactoringNoa
		subjectID = row.NoaID
	}

	customerName := stringutils.FirstNonEmpty(row.CustomerName, row.CustomerCode, "A customer")
	factorName := stringutils.FirstNonEmpty(row.FactorName, "its factor")

	title := customerName + " has no signed notice of assignment for " + factorName
	summary := "Their billing profile remits invoices to " + factorName +
		", but no notice has been drafted. Draft one and send it so the customer pays the factor."
	occurredAt := row.ProfileUpdatedAt
	if hasNotice {
		switch row.NoaStatus {
		case factoring.NoaStatusDraft:
			summary = "A notice to " + factorName + " is drafted but has not been sent. " +
				"Send it to the customer for signature."
			occurredAt = intutils.FirstPositive(
				typeutils.ValueOrZero(row.NoaCreatedAt),
				row.ProfileUpdatedAt,
			)
		case factoring.NoaStatusSent:
			summary = "The notice is waiting on the customer's signed copy"
			if row.NoaSentAt != nil {
				summary += "; it was sent on " + factoringDate(*row.NoaSentAt)
			}
			summary += ". Record the acknowledgement when it comes back."
			occurredAt = intutils.FirstPositive(
				typeutils.ValueOrZero(row.NoaSentAt),
				row.ProfileUpdatedAt,
			)
		case factoring.NoaStatusAcknowledged:
			title = customerName + "'s notice of assignment for " + factorName + " has expired"
			summary = "Release the expired notice and send a new one; until it is signed, " +
				"invoices to this customer are not ready to factor."
			if row.NoaExpiresAt != nil {
				summary = "The notice expired on " + factoringDate(
					*row.NoaExpiresAt,
				) + ". " + summary
				occurredAt = *row.NoaExpiresAt
			}
		case factoring.NoaStatusReleased:
			return watchtowerports.WatchtowerItemInput{}, false
		}
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourceFactoring,
		SourceID:    FactoringMissingNoaSourceID(row.CustomerID, row.FactoringCompanyID),
		Severity:    watchtower.SeverityWarning,
		Title:       stringutils.TruncateRunes(title, maxFactoringTitleLength),
		Summary:     stringutils.OneLine(summary, watchtoweritem.MaxSummaryLength),
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Path:        factoringRecordPath("customer", row.CustomerID, pathFactoringCustomers),
		OccurredAt:  intutils.FirstPositive(occurredAt, now),
	}, true
}

func DescribeFactoringNoaExpiring(
	tenant pagination.TenantInfo,
	row *factoringwatchtowerports.ExpiringNoa,
	now int64,
) (watchtowerports.WatchtowerItemInput, bool) {
	if row == nil || row.NoaID.IsNil() || row.ExpiresAt <= now ||
		row.ExpiresAt-now > FactoringNoaWarningSeconds() {
		return watchtowerports.WatchtowerItemInput{}, false
	}

	remaining := row.ExpiresAt - now
	severity := watchtower.SeverityWarning
	if remaining <= FactoringNoaCriticalWithin {
		severity = watchtower.SeverityCritical
	}

	customerName := stringutils.FirstNonEmpty(row.CustomerName, "A customer")
	factorName := stringutils.FirstNonEmpty(row.FactorName, "its factor")
	title := customerName + "'s notice of assignment for " + factorName + " expires " +
		remainingLabel(remaining)
	summary := "It expires on " + factoringDate(row.ExpiresAt) +
		". Get a renewed notice signed before then, or invoices to this customer " +
		"stop being ready to factor."

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourceFactoring,
		SourceID:    FactoringExpiringNoaSourceID(row.NoaID),
		Severity:    severity,
		Title:       stringutils.TruncateRunes(title, maxFactoringTitleLength),
		Summary:     stringutils.OneLine(summary, watchtoweritem.MaxSummaryLength),
		SubjectType: agent.SubjectFactoringNoa,
		SubjectID:   row.NoaID,
		EventKind:   agenttrigger.EventFactoringNoaExpiring,
		Path:        factoringRecordPath("customer", row.CustomerID, pathFactoringCustomers),
		OccurredAt: max(
			row.ExpiresAt-FactoringNoaWarningSeconds(),
			typeutils.ValueOrZero(row.AcknowledgedAt),
			1,
		),
	}, true
}

func factoringRecordPath(entity string, id pulid.ID, fallback string) string {
	if id.IsNotNil() {
		if recordPath, ok := productguide.RecordPath(entity, id.String()); ok {
			return recordPath
		}
	}
	return fallback
}

func factoringDate(ts int64) string {
	return time.Unix(ts, 0).UTC().Format(factoringDateLayout)
}

func waitedLabel(seconds int64) string {
	days := seconds / timeutils.SecondsPerDay
	if days < 2 {
		return "over a day"
	}
	return strconv.FormatInt(days, 10) + " days"
}

func remainingLabel(seconds int64) string {
	days := (seconds + timeutils.SecondsPerDay - 1) / timeutils.SecondsPerDay
	if days <= 1 {
		return "within a day"
	}
	return "in " + strconv.FormatInt(days, 10) + " days"
}

func FactoringRemittanceSourceID(remittanceID pulid.ID) string {
	return factoringRemittancePrefix + remittanceID.String()
}

func DescribeFactoringRemittanceExceptions(
	tenant pagination.TenantInfo,
	row *factoringremittanceports.OpenRemittance,
) (watchtowerports.WatchtowerItemInput, bool) {
	if row == nil || row.RemittanceID.IsNil() {
		return watchtowerports.WatchtowerItemInput{}, false
	}
	needsMapping := row.Status == factoringremittance.RemittanceStatusNeedsMapping
	if !needsMapping && (row.ExceptionCount <= 0 || !row.Status.CanEditLines()) {
		return watchtowerports.WatchtowerItemInput{}, false
	}

	factorName := stringutils.FirstNonEmpty(row.FactorName, "the factor")
	fileName := stringutils.FirstNonEmpty(row.FileName, "A remittance")
	title := fileName + " from " + factorName + " has lines to resolve"
	summary := strconv.Itoa(row.ExceptionCount) + " lines could not be matched on their own"
	if row.MatchedCount > 0 {
		summary += " and " + strconv.Itoa(row.MatchedCount) + " matched lines wait to be applied"
	}
	summary += ". Match or ignore each exception, then apply the remittance."
	if needsMapping {
		title = fileName + " from " + factorName + " needs its columns mapped"
		summary = "The file's headers did not say which columns hold the invoice and the " +
			"amounts. Map them once and the factor's next file is read on its own."
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourceFactoring,
		SourceID:    FactoringRemittanceSourceID(row.RemittanceID),
		Severity:    watchtower.SeverityWarning,
		Title:       stringutils.TruncateRunes(title, maxFactoringTitleLength),
		Summary:     stringutils.OneLine(summary, watchtoweritem.MaxSummaryLength),
		SubjectType: agent.SubjectFactoringRemittance,
		SubjectID:   row.RemittanceID,
		Path: factoringRecordPath(
			"factoring_remittance",
			row.RemittanceID,
			pathFactoringQueue,
		),
		OccurredAt: row.CreatedAt,
	}, true
}
