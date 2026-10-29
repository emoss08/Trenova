// Package creditreviewservice keeps every customer's credit standing in step
// with what it owes. Each time a customer's receivables ledger moves, its open
// balance is recorded on its billing profile and, where the profile asks for
// it, a hold is placed once the balance passes the credit limit and lifted
// once it is paid back under. A nightly sweep reviews everyone, so a limit
// changed on the form or a review that failed to run is caught up.
package creditreviewservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// LedgerStoreName tags the receivables ledger as stored, before the review
// wraps it; everything else receives the wrapped ledger.
const LedgerStoreName = `name:"customerLedgerStore"`

const (
	eventCreditHeld     = "customer_credit_held"
	eventCreditReleased = "customer_credit_released"
	noticeSource        = "creditreviewservice"
	maxNoticeRecipients = 25
	// SweepPageSize is how many customers the nightly sweep reviews per query.
	SweepPageSize = 200
	// realtimeResource must match the "customers" key the browser maps to its
	// customer queries.
	realtimeResource = "customers"
)

type permittedNotifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}

type ledgerBalances interface {
	SumCustomerBalances(
		ctx context.Context,
		req customerledgerports.SumCustomerBalancesRequest,
	) (map[pulid.ID]int64, error)
}

type creditStandings interface {
	ReviewCredit(
		ctx context.Context,
		req *customerports.ReviewCustomerCreditRequest,
	) (*customerports.ReviewCustomerCreditResult, error)
	ListCreditReviewCustomers(
		ctx context.Context,
		req *customerports.ListCreditReviewCustomersRequest,
	) ([]customerports.CreditReviewCustomer, error)
}

type Params struct {
	fx.In

	Ledger        customerledgerports.CustomerLedgerProjectionRepository `name:"customerLedgerStore"`
	Customers     customerports.CustomerRepository
	Notifications *notificationservice.Service  `                           optional:"true"`
	Realtime      realtimeports.RealtimeService `                           optional:"true"`
	Logger        *zap.Logger
}

type Service struct {
	ledger    ledgerBalances
	customers creditStandings
	notifier  permittedNotifier
	realtime  realtimeports.RealtimeService
	l         *zap.Logger
}

func New(p Params) *Service {
	s := &Service{
		ledger:    p.Ledger,
		customers: p.Customers,
		realtime:  p.Realtime,
		l:         p.Logger.Named("service.credit-review"),
	}
	if p.Notifications != nil {
		s.notifier = p.Notifications
	}
	return s
}

// Review brings the named customers' credit standing up to date with their
// receivables. Each customer is reviewed on its own, so one failure is
// reported without leaving the rest unreviewed.
func (s *Service) Review(
	ctx context.Context,
	tenant pagination.TenantInfo,
	customerIDs []pulid.ID,
) error {
	ids := uniqueIDs(customerIDs)
	if len(ids) == 0 {
		return nil
	}

	balances, err := s.ledger.SumCustomerBalances(
		ctx,
		customerledgerports.SumCustomerBalancesRequest{
			TenantInfo:                 tenant,
			CustomerIDs:                ids,
			ExcludeNonRecourseFactored: true,
		},
	)
	if err != nil {
		return fmt.Errorf("sum customer balances for credit review: %w", err)
	}

	now := timeutils.NowUnix()
	var failed []pulid.ID
	for _, id := range ids {
		result, reviewErr := s.customers.ReviewCredit(
			ctx,
			&customerports.ReviewCustomerCreditRequest{
				TenantInfo:    tenant,
				CustomerID:    id,
				ExposureMinor: max(balances[id], 0),
				Now:           now,
			},
		)
		if reviewErr != nil {
			s.l.Error("failed to review customer credit",
				zap.String("customerId", id.String()), zap.Error(reviewErr))
			failed = append(failed, id)
			continue
		}
		s.announce(ctx, tenant, id, result, now)
	}
	if len(failed) > 0 {
		return fmt.Errorf("credit review failed for %d of %d customers", len(failed), len(ids))
	}
	return nil
}

// SweepResult is what the nightly sweep covered.
type SweepResult struct {
	Customers int
	Failed    int
}

// Sweep reviews every customer with a billing profile, a page of customers at
// a time and each organization under its own tenant scope.
func (s *Service) Sweep(ctx context.Context, heartbeat func(string)) (*SweepResult, error) {
	result := new(SweepResult)
	var after *customerports.CreditReviewCustomer
	for {
		page, err := s.customers.ListCreditReviewCustomers(
			ctx,
			&customerports.ListCreditReviewCustomersRequest{
				After: after,
				Limit: SweepPageSize,
			},
		)
		if err != nil {
			return result, err
		}
		if len(page) == 0 {
			return result, nil
		}

		for _, group := range groupByTenant(page) {
			if heartbeat != nil {
				heartbeat(group.tenant.OrgID.String())
			}
			tenantCtx := dbscope.WithTenant(ctx, dbscope.Tenant{
				OrganizationID: group.tenant.OrgID,
				BusinessUnitID: group.tenant.BuID,
			})
			result.Customers += len(group.customerIDs)
			if reviewErr := s.Review(tenantCtx, group.tenant, group.customerIDs); reviewErr != nil {
				result.Failed += len(group.customerIDs)
				s.l.Error("credit sweep failed for an organization",
					zap.String("organizationId", group.tenant.OrgID.String()),
					zap.Error(reviewErr))
			}
		}

		last := page[len(page)-1]
		after = &last
		if len(page) < SweepPageSize {
			return result, nil
		}
	}
}

// announce tells the people who collect receivables when a customer goes on
// or comes off hold, and refreshes any screen showing the customer.
func (s *Service) announce(
	ctx context.Context,
	tenant pagination.TenantInfo,
	customerID pulid.ID,
	result *customerports.ReviewCustomerCreditResult,
	now int64,
) {
	if result == nil || result.Profile == nil || !result.Outcome.ChangesStanding() {
		return
	}

	if s.realtime != nil {
		if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Resource:       realtimeResource,
			Action:         "updated",
			RecordID:       customerID,
		}); err != nil {
			s.l.Warn("failed to publish customer credit change", zap.Error(err))
		}
	}

	if s.notifier == nil {
		return
	}
	note, ok := creditNotice(result)
	if !ok {
		return
	}
	correlation := customerID.String() + ":" + string(result.Outcome)
	note.CorrelationID = &correlation
	note.Source = noticeSource
	note.Data = map[string]any{
		"link":         "/accounting/ar/customer-ledger?customerId=" + customerID.String(),
		"customerId":   customerID.String(),
		"creditStatus": string(result.Profile.CreditStatus),
	}
	note.RelatedEntities = map[string]any{"customerId": customerID.String()}
	if _, err := s.notifier.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant:       tenant,
		Resource:     permission.ResourceCustomerPayment,
		Operation:    permission.OpCreate,
		Limit:        maxNoticeRecipients,
		Now:          now,
		Notification: note,
	}); err != nil {
		s.l.Warn("failed to notify about a customer credit change",
			zap.String("customerId", customerID.String()), zap.Error(err))
	}
}

// creditNotice words a hold placed or lifted. A warning moving on or off is
// left to the customer's screen rather than paging anyone.
func creditNotice(
	result *customerports.ReviewCustomerCreditResult,
) (notification.Notification, bool) {
	name := result.CustomerName
	if name == "" {
		name = result.CustomerCode
	}
	switch result.Outcome { //nolint:exhaustive // only a hold or a release is announced
	case customer.CreditReviewHeld:
		return notification.Notification{
			EventType: eventCreditHeld,
			Priority:  notification.PriorityHigh,
			Title:     name + " is on credit hold",
			Message: result.Profile.CreditHoldReason +
				". No new shipments can be booked for them until the balance is paid down or the hold is lifted.",
		}, true
	case customer.CreditReviewReleased:
		return notification.Notification{
			EventType: eventCreditReleased,
			Priority:  notification.PriorityMedium,
			Title:     name + " is off credit hold",
			Message:   "Their open receivables are back within the credit limit, so shipments can be booked for them again.",
		}, true
	default:
		return notification.Notification{}, false
	}
}

type tenantGroup struct {
	tenant      pagination.TenantInfo
	customerIDs []pulid.ID
}

func groupByTenant(page []customerports.CreditReviewCustomer) []tenantGroup {
	groups := make([]tenantGroup, 0, 1)
	for _, entry := range page {
		last := len(groups) - 1
		if last < 0 || groups[last].tenant.OrgID != entry.TenantInfo.OrgID ||
			groups[last].tenant.BuID != entry.TenantInfo.BuID {
			groups = append(groups, tenantGroup{tenant: entry.TenantInfo})
			last++
		}
		groups[last].customerIDs = append(groups[last].customerIDs, entry.CustomerID)
	}
	return groups
}

func uniqueIDs(ids []pulid.ID) []pulid.ID {
	out := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		if id.IsNil() || slices.Contains(out, id) {
			continue
		}
		out = append(out, id)
	}
	return out
}
