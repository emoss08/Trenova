package customertrackingservice

import (
	"context"
	"errors"
	"fmt"
	"html/template"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	dueBatchSize        = 100
	sweepWindowSeconds  = customertracking.NotificationLookbackHours * 3600
	linkPurgeAfterDays  = 30
	linkPurgeBatchLimit = 500
)

var (
	errEmailNotConfigured = errors.New("email delivery is not configured for this environment")
	errShipmentGone       = errors.New("the shipment no longer exists")
)

// shipmentContext is what delivering a shipment's notifications needs, read
// once per shipment rather than once per email.
type shipmentContext struct {
	tenant   pagination.TenantInfo
	shipment *shipment.Shipment
	customer *customer.Customer
	profile  *customertracking.Profile
	stops    []routeStop
}

func (s *Service) Evaluate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*customertrackingports.EvaluateResult, error) {
	result := &customertrackingports.EvaluateResult{}
	sc, err := s.shipmentContext(ctx, tenantInfo, shipmentID)
	if errors.Is(err, errShipmentGone) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if !sc.profile.NotifiesAny() {
		return result, nil
	}

	facts, err := s.shipmentFacts(ctx, sc)
	if err != nil {
		return result, err
	}
	now := s.now()
	occurrences := customertracking.Detect(facts, sc.profile, now)
	result.Detected = len(occurrences)
	for _, occurrence := range occurrences {
		row := &customertracking.Notification{
			ID:             customertracking.NewNotificationID(),
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			ShipmentID:     sc.shipment.ID,
			CustomerID:     sc.shipment.CustomerID,
			Event:          occurrence.Event,
			SubjectKey:     occurrence.SubjectKey,
			StopID:         occurrence.StopID,
			OccurredAt:     occurrence.OccurredAt,
			EstimatedAt:    occurrence.EstimatedAt,
			Status:         customertracking.NotificationStatusPending,
			Recipients:     []string{},
			NextAttemptAt:  now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		inserted, insertErr := s.notifications.InsertOnce(ctx, row)
		if insertErr != nil {
			return result, insertErr
		}
		if !inserted {
			continue
		}
		result.Recorded++
		s.tally(result, s.deliver(ctx, sc, row))
	}
	return result, nil
}

func (s *Service) shipmentContext(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*shipmentContext, error) {
	sh, err := s.loadShipment(ctx, tenantInfo, shipmentID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errShipmentGone
		}
		return nil, err
	}
	profile, err := s.profiles.GetByCustomer(ctx, tenantInfo, sh.CustomerID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		profile = customertracking.DefaultProfile(tenantInfo.OrgID, tenantInfo.BuID, sh.CustomerID)
	}
	cus, err := s.loadCustomer(ctx, tenantInfo, sh.CustomerID)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	return &shipmentContext{
		tenant:   tenantInfo,
		shipment: sh,
		customer: cus,
		profile:  profile,
		stops:    route(sh),
	}, nil
}

func (s *Service) shipmentFacts(
	ctx context.Context,
	sc *shipmentContext,
) (*customertracking.ShipmentFacts, error) {
	sh := sc.shipment
	facts := &customertracking.ShipmentFacts{
		ShipmentID: sh.ID,
		CustomerID: sh.CustomerID,
		Canceled:   sh.IsCanceled(),
		Completed:  isClosed(sh) && !sh.IsCanceled(),
		Stops:      make([]customertracking.StopFacts, 0, len(sc.stops)),
	}
	for _, rs := range sc.stops {
		facts.Stops = append(facts.Stops, customertracking.StopFacts{
			ID:              rs.stop.ID,
			Pickup:          rs.stop.IsOriginStop(),
			Delivery:        rs.stop.IsDestinationStop(),
			ActualArrival:   rs.stop.ActualArrival,
			ActualDeparture: rs.stop.ActualDeparture,
		})
	}
	if sc.profile.NotifyDelayed {
		etas, err := s.stopEtas.ListByShipmentIDs(ctx, sc.tenant, []pulid.ID{sh.ID})
		if err != nil {
			return nil, err
		}
		for _, eta := range etas {
			facts.Etas = append(facts.Etas, customertracking.EtaFacts{
				StopID:           eta.StopID,
				NeedsAttention:   eta.Verdict.NeedsAttention(),
				Arrived:          eta.Arrived,
				EstimatedArrival: eta.EstimatedArrival,
				ComputedAt:       eta.ComputedAt,
			})
		}
	}
	if sc.profile.NotifyPodReady {
		pods, err := s.facts.LatestPods(ctx, sc.tenant, []pulid.ID{sh.ID})
		if err != nil {
			return nil, err
		}
		if pod := pods[sh.ID]; pod != nil {
			facts.Pod = &customertracking.PodFacts{DocumentID: pod.DocumentID, FiledAt: pod.FiledAt}
		}
	}
	return facts, nil
}

func (s *Service) tally(
	result *customertrackingports.EvaluateResult,
	status customertracking.NotificationStatus,
) {
	switch status {
	case customertracking.NotificationStatusSent:
		result.Delivered++
	case customertracking.NotificationStatusSkipped:
		result.Skipped++
	case customertracking.NotificationStatusFailed:
		result.Failed++
	case customertracking.NotificationStatusPending:
	}
}

// recipients is who a customer's status emails go to: the addresses on its
// tracking settings, else its status update recipients, else the To line of
// its email profile.
func recipients(sc *shipmentContext) []string {
	if list := sc.profile.RecipientList(); len(list) > 0 {
		return list
	}
	if sc.customer == nil {
		return nil
	}
	if list := stringutils.SplitEmailList(sc.customer.StatusUpdateRecipients); len(list) > 0 {
		return list
	}
	if sc.customer.EmailProfile != nil {
		return stringutils.SplitEmailList(sc.customer.EmailProfile.ToRecipients)
	}
	return nil
}

// deliver sends one owed email and settles its ledger row. A failure leaves
// the row pending with a backed-off retry time until it runs out of attempts.
// Whatever it returns, the row is never sent twice: the email goes out under
// the row's idempotency key, and the row is settled only while still pending.
func (s *Service) deliver(
	ctx context.Context,
	sc *shipmentContext,
	row *customertracking.Notification,
) customertracking.NotificationStatus {
	now := s.now()
	outcome := &customertrackingports.DeliveryOutcome{
		TenantInfo:    sc.tenant,
		ID:            row.ID,
		Recipients:    row.Recipients,
		LinkID:        row.LinkID,
		Attempts:      row.Attempts,
		NextAttemptAt: row.NextAttemptAt,
		At:            now,
	}

	switch {
	case sc.shipment.IsCanceled():
		outcome.Status, outcome.SkipReason = customertracking.NotificationStatusSkipped,
			customertracking.SkipReasonShipmentGone
	case !sc.profile.Notifies(row.Event):
		outcome.Status, outcome.SkipReason = customertracking.NotificationStatusSkipped,
			customertracking.SkipReasonRuleOff
	default:
		to := recipients(sc)
		if len(to) == 0 {
			outcome.Status, outcome.SkipReason = customertracking.NotificationStatusSkipped,
				customertracking.SkipReasonNoRecipients
			break
		}
		outcome.Recipients = to
		message, linkID, err := s.send(ctx, sc, row, to)
		outcome.LinkID = linkID
		if err != nil {
			outcome.Attempts = row.Attempts + 1
			outcome.LastError = truncate(err.Error(), customertracking.MaxLastErrorLength)
			if outcome.Attempts >= customertracking.MaxDeliveryAttempts {
				outcome.Status = customertracking.NotificationStatusFailed
			} else {
				outcome.Status = customertracking.NotificationStatusPending
				outcome.NextAttemptAt = now + customertracking.RetryDelaySeconds(outcome.Attempts)
			}
			s.l.Warn("failed to send customer status email",
				zap.String("notificationId", row.ID.String()),
				zap.String("event", string(row.Event)),
				zap.Int16("attempts", outcome.Attempts),
				zap.Error(err))
			break
		}
		outcome.Status = customertracking.NotificationStatusSent
		outcome.Attempts = row.Attempts + 1
		outcome.SentAt = &now
		if message != nil {
			outcome.EmailMessageID = message.ID
		}
	}

	settled, err := s.notifications.Record(ctx, outcome)
	if err != nil {
		s.l.Error("failed to record customer status email outcome",
			zap.String("notificationId", row.ID.String()),
			zap.Error(err))
		return customertracking.NotificationStatusPending
	}
	if !settled {
		return customertracking.NotificationStatusPending
	}
	return outcome.Status
}

func (s *Service) send(
	ctx context.Context,
	sc *shipmentContext,
	row *customertracking.Notification,
	to []string,
) (*email.Message, pulid.ID, error) {
	if !s.canEmail() {
		return nil, row.LinkID, errEmailNotConfigured
	}
	data := s.statusEmailContext(ctx, sc, row)
	linkID := row.LinkID
	if sc.profile.IncludeLink && sc.profile.LinkEnabled {
		minted, err := s.mintLink(
			ctx,
			sc.shipment,
			sc.profile,
			customertracking.LinkPurposeNotification,
			to[0],
			pulid.Nil,
		)
		if err != nil {
			s.l.Warn("customer status email goes without a tracking link",
				zap.String("notificationId", row.ID.String()),
				zap.Error(err))
		} else {
			linkID = minted.link.ID
			//nolint:gosec // G203: an operator-configured base URL and a server-minted, path-escaped token
			data.TrackingURL = template.URL(minted.url)
			data.ExpiresAt = timeutils.WindowLabelUTC(minted.link.ExpiresAt, nil)
		}
	}

	customerID := sc.shipment.CustomerID
	rendered, err := s.templates.RenderMessage(ctx, &documenttemplateports.RenderMessageRequest{
		TenantInfo:        sc.tenant,
		Kind:              documenttemplate.KindCustomerStatusEmail,
		CustomerID:        &customerID,
		Data:              data,
		ReferenceID:       row.ID,
		FallbackToBuiltIn: true,
	})
	if err != nil {
		return nil, linkID, fmt.Errorf("render customer status email: %w", err)
	}
	message, err := s.emailService.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo:     sc.tenant,
		Purpose:        email.PurposeOperations,
		To:             to,
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: row.IdempotencyKey(),
	})
	if err != nil {
		return nil, linkID, fmt.Errorf("send customer status email: %w", err)
	}
	return message, linkID, nil
}

func (s *Service) statusEmailContext(
	ctx context.Context,
	sc *shipmentContext,
	row *customertracking.Notification,
) documenttemplate.CustomerStatusEmailContext {
	sh := sc.shipment
	origin, destination := originAndDestination(sc.stops)
	data := documenttemplate.CustomerStatusEmailContext{
		CompanyName:       s.companyName(ctx, sc.tenant),
		Event:             string(row.Event),
		IsPickedUp:        row.Event == customertracking.EventPickedUp,
		IsDelayed:         row.Event == customertracking.EventDelayed,
		IsDelivered:       row.Event == customertracking.EventDelivered,
		IsPodReady:        row.Event == customertracking.EventPodReady,
		ShipmentProNumber: sh.ProNumber,
		BOL:               sh.BOL,
	}
	if sc.customer != nil {
		data.CustomerName = sc.customer.Name
	}
	if origin != nil {
		data.OriginSummary = placeSummary(origin)
	}
	if destination != nil {
		data.DestinationSummary = placeSummary(destination)
	}

	anchor := destination
	switch row.Event {
	case customertracking.EventPickedUp:
		anchor = origin
	case customertracking.EventDelayed:
		if stop := findStop(sc.stops, row.StopID); stop != nil {
			anchor = stop
			data.StopSummary = placeSummary(stop)
			data.StopWindow = stopWindowLabel(stop)
		}
		if row.EstimatedAt != nil {
			data.EstimatedArrival = instantLabel(*row.EstimatedAt, anchor)
		}
	case customertracking.EventDelivered, customertracking.EventPodReady:
	}
	data.OccurredAt = instantLabel(row.OccurredAt, anchor)
	return data
}

func findStop(stops []routeStop, stopID pulid.ID) *shipment.Stop {
	if stopID.IsNil() {
		return nil
	}
	for _, rs := range stops {
		if rs.stop.ID == stopID {
			return rs.stop
		}
	}
	return nil
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

// DeliverDue retries the organization's pending emails whose time has come.
func (s *Service) DeliverDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*customertrackingports.EvaluateResult, error) {
	result := &customertrackingports.EvaluateResult{}
	rows, err := s.notifications.ListDue(ctx, tenantInfo, s.now(), dueBatchSize)
	if err != nil {
		return result, err
	}
	contexts := make(map[pulid.ID]*shipmentContext, len(rows))
	for _, row := range rows {
		sc, ok := contexts[row.ShipmentID]
		if !ok {
			sc, err = s.shipmentContext(ctx, tenantInfo, row.ShipmentID)
			if err != nil && !errors.Is(err, errShipmentGone) {
				return result, err
			}
			contexts[row.ShipmentID] = sc
		}
		if sc == nil {
			s.tally(result, s.settleGone(ctx, tenantInfo, row))
			continue
		}
		s.tally(result, s.deliver(ctx, sc, row))
	}
	return result, nil
}

func (s *Service) settleGone(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	row *customertracking.Notification,
) customertracking.NotificationStatus {
	settled, err := s.notifications.Record(ctx, &customertrackingports.DeliveryOutcome{
		TenantInfo:    tenantInfo,
		ID:            row.ID,
		Status:        customertracking.NotificationStatusSkipped,
		SkipReason:    customertracking.SkipReasonShipmentGone,
		Recipients:    row.Recipients,
		Attempts:      row.Attempts,
		NextAttemptAt: row.NextAttemptAt,
		At:            s.now(),
	})
	if err != nil || !settled {
		return customertracking.NotificationStatusPending
	}
	return customertracking.NotificationStatusSkipped
}

// SweepTenant evaluates every shipment in the organization that could owe its
// customer an email, then retries what is still pending. It is the safety net
// under the per-shipment evaluations, and the only path a POD filed without a
// shipment event takes.
func (s *Service) SweepTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*customertrackingports.SweepResult, error) {
	result := &customertrackingports.SweepResult{}
	ids, err := s.facts.ListCandidateShipments(
		ctx,
		&customertrackingports.CandidateShipmentsRequest{
			TenantInfo: tenantInfo,
			Since:      s.now() - sweepWindowSeconds,
		},
	)
	if err != nil {
		return result, err
	}
	result.Shipments = len(ids)
	for _, id := range ids {
		evaluated, evalErr := s.Evaluate(ctx, tenantInfo, id)
		if evalErr != nil {
			s.l.Warn("customer status sweep could not evaluate a shipment",
				zap.String("shipmentId", id.String()),
				zap.Error(evalErr))
			continue
		}
		addResult(&result.EvaluateResult, evaluated)
	}
	due, err := s.DeliverDue(ctx, tenantInfo)
	if err != nil {
		return result, err
	}
	addResult(&result.EvaluateResult, due)
	return result, nil
}

func addResult(total, part *customertrackingports.EvaluateResult) {
	if part == nil {
		return
	}
	total.Detected += part.Detected
	total.Recorded += part.Recorded
	total.Delivered += part.Delivered
	total.Skipped += part.Skipped
	total.Failed += part.Failed
}

func (s *Service) ListSweepTenants(ctx context.Context) ([]pagination.TenantInfo, error) {
	notifying, err := s.facts.ListNotifyingTenants(ctx)
	if err != nil {
		return nil, err
	}
	due, err := s.notifications.ListTenantsWithDue(ctx, s.now())
	if err != nil {
		return nil, err
	}
	seen := make(map[pagination.TenantInfo]struct{}, len(notifying)+len(due))
	tenants := make([]pagination.TenantInfo, 0, len(notifying)+len(due))
	for _, tenant := range append(notifying, due...) {
		key := pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tenants = append(tenants, key)
	}
	return tenants, nil
}

func (s *Service) PurgeDeadLinks(ctx context.Context) (int, error) {
	before := s.now() - linkPurgeAfterDays*customertracking.SecondsPerDay
	return s.links.PurgeDead(ctx, before, linkPurgeBatchLimit)
}
