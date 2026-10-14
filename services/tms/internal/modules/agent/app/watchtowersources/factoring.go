package watchtowersources

import (
	"context"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
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
	FactoringReadyCriticalAfter = int64(3 * timeutils.SecondsPerDay)
	FactoringNoaCriticalWithin  = int64(7 * timeutils.SecondsPerDay)
	pathFactoringInvoices       = "/billing/invoices"
	pathFactoringCustomers      = "/billing/configuration-files/customers"
	factoringDateLayout         = "January 2, 2006"
	maxFactoringTitleLength     = 200
)

type FactoringSource struct {
	states factoringports.InvoiceStateRepository
	reader factoringports.WatchtowerReader
	now    func() int64
}

func NewFactoringSource(
	states factoringports.InvoiceStateRepository,
	reader factoringports.WatchtowerReader,
) watchtowerports.WatchtowerSource {
	return &FactoringSource{states: states, reader: reader, now: timeutils.NowUnix}
}

func (s *FactoringSource) Kind() watchtower.SourceKind { return watchtower.SourceFactoring }

func (s *FactoringSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	now := s.now()

	ready, err := s.states.ListReadyUnsubmitted(ctx, factoringports.ListReadyUnsubmittedRequest{
		TenantInfo:  tenant,
		ReadyBefore: now - factoring.ReadyUnsubmittedSeconds,
		Limit:       snapshotLimit,
	})
	if err != nil {
		return nil, err
	}

	invoiceIDs := make([]pulid.ID, 0, len(ready))
	for _, state := range ready {
		if state != nil {
			invoiceIDs = append(invoiceIDs, state.InvoiceID)
		}
	}
	labels, err := s.reader.ListInvoiceLabels(ctx, &factoringports.ListInvoiceLabelsRequest{
		TenantInfo: tenant,
		InvoiceIDs: invoiceIDs,
	})
	if err != nil {
		return nil, err
	}

	missing, err := s.reader.ListCustomersMissingNoa(
		ctx,
		&factoringports.ListCustomersMissingNoaRequest{
			TenantInfo: tenant,
			At:         now,
			Limit:      snapshotLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	expiring, err := s.reader.ListNoasExpiringSoon(ctx, &factoringports.ListNoasExpiringSoonRequest{
		TenantInfo: tenant,
		At:         now,
		Before:     now + FactoringNoaWarningSeconds(),
		Limit:      snapshotLimit,
	})
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(ready)+len(missing)+len(expiring))
	for _, state := range ready {
		if state == nil {
			continue
		}
		if item, open := DescribeFactoringReadyUnsubmitted(
			tenant,
			state,
			labels[state.InvoiceID],
			now,
		); open {
			items = append(items, item)
		}
	}
	for _, row := range missing {
		if item, open := DescribeFactoringMissingNoa(tenant, row, now); open {
			items = append(items, item)
		}
	}
	for _, row := range expiring {
		if item, open := DescribeFactoringNoaExpiring(tenant, row, now); open {
			items = append(items, item)
		}
	}

	return items, nil
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

func DescribeFactoringReadyUnsubmitted(
	tenant pagination.TenantInfo,
	state *factoring.InvoiceState,
	label *factoringports.InvoiceLabel,
	now int64,
) (watchtowerports.WatchtowerItemInput, bool) {
	if state == nil || state.InvoiceID.IsNil() || state.ReadySince == nil ||
		!state.ReadyLongerThan(now, factoring.ReadyUnsubmittedSeconds) {
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
		Summary:     stringutils.OneLine(summary, maxSummaryLength),
		SubjectType: agent.SubjectInvoice,
		SubjectID:   state.InvoiceID,
		EventKind:   agent.EventFactoringInvoiceReady,
		Path:        factoringRecordPath("invoice", state.InvoiceID, pathFactoringInvoices),
		OccurredAt:  *state.ReadySince + factoring.ReadyUnsubmittedSeconds,
	}, true
}

func DescribeFactoringMissingNoa(
	tenant pagination.TenantInfo,
	row *factoringports.CustomerMissingNoa,
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
			occurredAt = firstNonZero(typeutils.ValueOrZero(row.NoaCreatedAt), row.ProfileUpdatedAt)
		case factoring.NoaStatusSent:
			summary = "The notice is waiting on the customer's signed copy"
			if row.NoaSentAt != nil {
				summary += "; it was sent on " + factoringDate(*row.NoaSentAt)
			}
			summary += ". Record the acknowledgement when it comes back."
			occurredAt = firstNonZero(typeutils.ValueOrZero(row.NoaSentAt), row.ProfileUpdatedAt)
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
		Summary:     stringutils.OneLine(summary, maxSummaryLength),
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Path:        factoringRecordPath("customer", row.CustomerID, pathFactoringCustomers),
		OccurredAt:  firstNonZero(occurredAt, now),
	}, true
}

func DescribeFactoringNoaExpiring(
	tenant pagination.TenantInfo,
	row *factoringports.ExpiringNoa,
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
		Summary:     stringutils.OneLine(summary, maxSummaryLength),
		SubjectType: agent.SubjectFactoringNoa,
		SubjectID:   row.NoaID,
		EventKind:   agent.EventFactoringNoaExpiring,
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
