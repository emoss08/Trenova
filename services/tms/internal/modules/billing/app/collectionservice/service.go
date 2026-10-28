// Package collectionservice runs collections: the ranked worklist of
// customers with overdue receivables, and the history, promises, collector
// and follow-ups each account is worked with.
package collectionservice

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// DefaultWorklistLimit and MaxWorklistLimit bound a worklist page.
	DefaultWorklistLimit = 50
	MaxWorklistLimit     = 200
	// ActivityPageSize is how much history the account panel loads at once.
	ActivityPageSize = 50
	// realtimeResource must match the "collection-account" key the browser
	// maps to its collections queries.
	realtimeResource     = "collection-account"
	eventAccountAssigned = "collections_account_assigned"
	secondsPerDay        = 24 * 60 * 60
)

type notifier interface {
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
}

type members interface {
	GetTenantMember(
		ctx context.Context,
		req tenantports.GetTenantMemberRequest,
	) (*tenant.User, error)
}

type Params struct {
	fx.In

	DB            kernelports.Transactor
	Repo          collectionports.Repository
	Customers     customerports.CustomerRepository
	Users         tenantports.UserRepository
	Notifications *notificationservice.Service  `optional:"true"`
	Realtime      realtimeports.RealtimeService `optional:"true"`
	Logger        *zap.Logger
}

type Service struct {
	db        kernelports.Transactor
	repo      collectionports.Repository
	customers customerports.CustomerRepository
	users     members
	notifier  notifier
	realtime  realtimeports.RealtimeService
	now       func() int64
	l         *zap.Logger
}

var _ collectionports.Service = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	s := &Service{
		db:        p.DB,
		repo:      p.Repo,
		customers: p.Customers,
		users:     p.Users,
		realtime:  p.Realtime,
		now:       timeutils.NowUnix,
		l:         p.Logger.Named("service.collection"),
	}
	if p.Notifications != nil {
		s.notifier = p.Notifications
	}
	return s
}

// Worklist ranks every account in scope, so the first page is always the
// customers to call first, and totals the whole list for the strip above it.
func (s *Service) Worklist(
	ctx context.Context,
	req *collectionports.WorklistRequest,
) (*collectionports.WorklistPage, error) {
	if req.Scope == "" {
		req.Scope = collectionports.CollectorScopeAll
	}
	if !req.Scope.IsValid() {
		return nil, errortypes.NewValidationError(
			"scope",
			errortypes.ErrInvalid,
			"Collector filter is invalid",
		)
	}
	req.Limit = clampLimit(req.Limit)
	req.Offset = max(req.Offset, 0)
	req.AsOf = s.now()

	page, err := s.repo.ListWorklist(ctx, req)
	if err != nil {
		return nil, err
	}

	rows := make([]*collectionports.WorklistRow, 0, len(page.Rows))
	summary := collectionports.WorklistSummary{}
	overdue := make(map[string]int64, 1)
	for _, row := range page.Rows {
		rank(row, req.AsOf)
		if req.DueOnly && !isDue(row) {
			continue
		}
		rows = append(rows, row)
		summary.Accounts++
		if row.OverdueMinor > 0 {
			overdue[row.CurrencyCode] += row.OverdueMinor
		}
		if row.FollowUpDue {
			summary.FollowUpsDue++
		}
		if row.Promise != nil && row.Promise.Status == collection.PromiseBroken {
			summary.BrokenPromises++
		}
		if row.CollectorID.IsNil() {
			summary.Unassigned++
		}
	}
	summary.Overdue = currencyTotals(overdue)
	slices.SortStableFunc(rows, func(a, b *collectionports.WorklistRow) int {
		return cmp.Or(
			cmp.Compare(a.Priority.Rank(), b.Priority.Rank()),
			cmp.Compare(b.OverdueMinor, a.OverdueMinor),
			cmp.Compare(b.OldestDaysPastDue, a.OldestDaysPastDue),
			strings.Compare(a.CustomerName, b.CustomerName),
		)
	})

	total := len(rows)
	start := min(req.Offset, total)
	end := min(start+req.Limit, total)
	return &collectionports.WorklistPage{
		Rows:    rows[start:end],
		Total:   total,
		Summary: summary,
	}, nil
}

// GetAccount is the customer as the collections panel shows it, with a page
// of history ending before the cursor.
func (s *Service) GetAccount(
	ctx context.Context,
	req collectionports.GetAccountRequest,
	before *collectionports.ActivityCursor,
) (*collectionports.AccountView, error) {
	now := s.now()
	row, err := s.repo.GetWorklistRow(ctx, req, now)
	if err != nil {
		return nil, err
	}
	rank(row, now)

	activities, err := s.repo.ListActivities(ctx, collectionports.ListActivitiesRequest{
		TenantInfo: req.TenantInfo,
		CustomerID: req.CustomerID,
		Before:     before,
		Limit:      ActivityPageSize + 1,
	})
	if err != nil {
		return nil, err
	}
	hasMore := len(activities) > ActivityPageSize
	if hasMore {
		activities = activities[:ActivityPageSize]
	}

	return &collectionports.AccountView{Row: row, Activities: activities, HasMore: hasMore}, nil
}

// LogActivity records a note, call, email or promise to pay. A promise
// becomes the account's standing promise and, unless the person scheduled
// otherwise, the account's follow-up moves to the day after it falls due.
func (s *Service) LogActivity(
	ctx context.Context,
	req *collectionports.LogActivityRequest,
	actor *permissionports.RequestActor,
) (*collectionports.AccountView, error) {
	activity, err := s.contactActivity(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		account, txErr := s.repo.GetAccountForUpdate(
			txCtx,
			accountRequest(req.TenantInfo, req.CustomerID),
		)
		if txErr != nil {
			return txErr
		}
		if txErr = s.repo.InsertActivity(txCtx, activity); txErr != nil {
			return txErr
		}
		if activity.IsPromise() {
			account.PromiseActivityID = activity.ID
		}
		account.NextFollowUpOn = followUpAfter(account.NextFollowUpOn, activity, req.FollowUpOn)
		_, txErr = s.repo.SaveAccount(txCtx, account)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.publish(ctx, req.TenantInfo, req.CustomerID, actor)
	return s.GetAccount(ctx, accountRequest(req.TenantInfo, req.CustomerID), nil)
}

func (s *Service) PlanLogActivity(
	ctx context.Context,
	req *collectionports.LogActivityRequest,
	actor *permissionports.RequestActor,
) (*collectionports.AccountChangePlan, error) {
	activity, err := s.contactActivity(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	plan, err := s.planFor(ctx, req.TenantInfo, req.CustomerID)
	if err != nil {
		return nil, err
	}

	plan.Activity = activity
	plan.Changed = true
	if activity.IsPromise() {
		plan.After.PromisedOn = activity.PromisedOn
		plan.After.PromisedAmountMinor = activity.PromisedAmountMinor
	}
	plan.After.NextFollowUpOn = followUpAfter(plan.Before.NextFollowUpOn, activity, req.FollowUpOn)
	return plan, nil
}

func (s *Service) contactActivity(
	ctx context.Context,
	req *collectionports.LogActivityRequest,
	actor *permissionports.RequestActor,
) (*collection.Activity, error) {
	if !req.Kind.IsContact() {
		return nil, errortypes.NewValidationError(
			"kind", errortypes.ErrInvalid, "Log a note, call, email or promise to pay",
		)
	}
	currency, err := s.billingCurrency(ctx, req.TenantInfo, req.CustomerID)
	if err != nil {
		return nil, err
	}

	activity := &collection.Activity{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		CustomerID:     req.CustomerID,
		Kind:           req.Kind,
		Summary:        req.Summary,
		ContactName:    req.ContactName,
		InvoiceID:      req.InvoiceID,
		CurrencyCode:   currency,
		FollowUpOn:     req.FollowUpOn,
		CreatedByID:    actorUserID(actor, req.TenantInfo),
	}
	if req.Kind == collection.ActivityPromiseToPay {
		if req.PromisedAmount.Valid {
			amount := money.MinorUnits(req.PromisedAmount.Decimal)
			activity.PromisedAmountMinor = &amount
		}
		activity.PromisedOn = req.PromisedOn
	}

	multiErr := errortypes.NewMultiError()
	activity.Validate(multiErr)
	for _, fieldErr := range multiErr.Errors {
		if fieldErr != nil && fieldErr.Field == "promisedAmountMinor" {
			fieldErr.Field = "promisedAmount"
		}
	}
	s.validateFollowUp(multiErr, "followUpOn", req.FollowUpOn)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return activity, nil
}

func followUpAfter(current *int64, activity *collection.Activity, scheduled *int64) *int64 {
	next := current
	if activity.IsPromise() {
		dayAfter := *activity.PromisedOn + secondsPerDay
		next = &dayAfter
	}
	if scheduled != nil {
		next = scheduled
	}
	return next
}

// AssignCollector gives the account to a member of the organization, or to
// nobody, records the change in the history, and tells the new collector.
func (s *Service) AssignCollector(
	ctx context.Context,
	req *collectionports.AssignCollectorRequest,
	actor *permissionports.RequestActor,
) (*collectionports.AccountView, error) {
	currency, err := s.billingCurrency(ctx, req.TenantInfo, req.CustomerID)
	if err != nil {
		return nil, err
	}
	collectorName, err := s.collectorName(ctx, req)
	if err != nil {
		return nil, err
	}

	actorID := actorUserID(actor, req.TenantInfo)
	changed := false
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		account, txErr := s.repo.GetAccountForUpdate(
			txCtx,
			accountRequest(req.TenantInfo, req.CustomerID),
		)
		if txErr != nil {
			return txErr
		}
		if account.CollectorID == req.CollectorID {
			return nil
		}
		changed = true
		account.CollectorID = req.CollectorID
		if _, txErr = s.repo.SaveAccount(txCtx, account); txErr != nil {
			return txErr
		}
		return s.repo.InsertActivity(txCtx, collectorActivity(req, collectorName, actorID))
	})
	if err != nil {
		return nil, err
	}

	view, err := s.GetAccount(ctx, accountRequest(req.TenantInfo, req.CustomerID), nil)
	if err != nil {
		return nil, err
	}
	if changed {
		s.publish(ctx, req.TenantInfo, req.CustomerID, actor)
		if req.CollectorID.IsNotNil() && req.CollectorID != actorID {
			s.tellCollector(ctx, req.TenantInfo, req.CollectorID, view.Row, currency)
		}
	}
	return view, nil
}

func (s *Service) PlanAssignCollector(
	ctx context.Context,
	req *collectionports.AssignCollectorRequest,
	actor *permissionports.RequestActor,
) (*collectionports.AccountChangePlan, error) {
	if _, err := s.billingCurrency(ctx, req.TenantInfo, req.CustomerID); err != nil {
		return nil, err
	}
	collectorName, err := s.collectorName(ctx, req)
	if err != nil {
		return nil, err
	}
	plan, err := s.planFor(ctx, req.TenantInfo, req.CustomerID)
	if err != nil {
		return nil, err
	}

	plan.After.CollectorID = req.CollectorID
	plan.After.CollectorName = collectorName
	if plan.Before.CollectorID == req.CollectorID {
		return plan, nil
	}
	actorID := actorUserID(actor, req.TenantInfo)
	plan.Changed = true
	plan.Activity = collectorActivity(req, collectorName, actorID)
	plan.NotifiesCollector = req.CollectorID.IsNotNil() && req.CollectorID != actorID &&
		s.notifier != nil
	return plan, nil
}

func (s *Service) collectorName(
	ctx context.Context,
	req *collectionports.AssignCollectorRequest,
) (string, error) {
	if req.CollectorID.IsNil() {
		return "", nil
	}
	member, err := s.users.GetTenantMember(ctx, tenantports.GetTenantMemberRequest{
		UserID:     req.CollectorID,
		TenantInfo: req.TenantInfo,
		Now:        s.now(),
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return "", errortypes.NewValidationError(
				"collectorId", errortypes.ErrInvalid, "Choose someone in this organization",
			)
		}
		return "", err
	}
	return member.Name, nil
}

func collectorActivity(
	req *collectionports.AssignCollectorRequest,
	collectorName string,
	actorID pulid.ID,
) *collection.Activity {
	summary := "Unassigned the collector"
	if req.CollectorID.IsNotNil() {
		summary = "Assigned to " + collectorName
	}
	return &collection.Activity{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		CustomerID:     req.CustomerID,
		Kind:           collection.ActivityCollectorAssigned,
		Summary:        summary,
		CollectorID:    req.CollectorID,
		CurrencyCode:   money.DefaultCurrencyCode,
		CreatedByID:    actorID,
	}
}

// ScheduleFollowUp sets when the account is next chased, or clears it, and
// records the change in the history.
func (s *Service) ScheduleFollowUp(
	ctx context.Context,
	req *collectionports.ScheduleFollowUpRequest,
	actor *permissionports.RequestActor,
) (*collectionports.AccountView, error) {
	activity, err := s.followUpActivity(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		account, txErr := s.repo.GetAccountForUpdate(
			txCtx,
			accountRequest(req.TenantInfo, req.CustomerID),
		)
		if txErr != nil {
			return txErr
		}
		account.NextFollowUpOn = req.FollowUpOn
		if _, txErr = s.repo.SaveAccount(txCtx, account); txErr != nil {
			return txErr
		}
		return s.repo.InsertActivity(txCtx, activity)
	})
	if err != nil {
		return nil, err
	}

	s.publish(ctx, req.TenantInfo, req.CustomerID, actor)
	return s.GetAccount(ctx, accountRequest(req.TenantInfo, req.CustomerID), nil)
}

func (s *Service) PlanScheduleFollowUp(
	ctx context.Context,
	req *collectionports.ScheduleFollowUpRequest,
	actor *permissionports.RequestActor,
) (*collectionports.AccountChangePlan, error) {
	activity, err := s.followUpActivity(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	plan, err := s.planFor(ctx, req.TenantInfo, req.CustomerID)
	if err != nil {
		return nil, err
	}

	plan.Activity = activity
	plan.Changed = true
	plan.After.NextFollowUpOn = req.FollowUpOn
	return plan, nil
}

func (s *Service) followUpActivity(
	ctx context.Context,
	req *collectionports.ScheduleFollowUpRequest,
	actor *permissionports.RequestActor,
) (*collection.Activity, error) {
	if _, err := s.billingCurrency(ctx, req.TenantInfo, req.CustomerID); err != nil {
		return nil, err
	}
	note := strings.TrimSpace(req.Note)
	multiErr := errortypes.NewMultiError()
	s.validateFollowUp(multiErr, "followUpOn", req.FollowUpOn)
	if len(note) > collection.MaxSummaryLength {
		multiErr.Add("note", errortypes.ErrInvalid, "Keep the note under 4,000 characters")
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	summary := note
	if summary == "" && req.FollowUpOn == nil {
		summary = "Cleared the follow-up"
	}
	return &collection.Activity{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		CustomerID:     req.CustomerID,
		Kind:           collection.ActivityFollowUpScheduled,
		Summary:        summary,
		FollowUpOn:     req.FollowUpOn,
		CurrencyCode:   money.DefaultCurrencyCode,
		CreatedByID:    actorUserID(actor, req.TenantInfo),
	}, nil
}

func (s *Service) planFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*collectionports.AccountChangePlan, error) {
	now := s.now()
	row, err := s.repo.GetWorklistRow(ctx, accountRequest(tenantInfo, customerID), now)
	if err != nil {
		return nil, err
	}
	rank(row, now)

	before := collectionports.AccountState{
		CollectorID:    row.CollectorID,
		CollectorName:  row.CollectorName,
		NextFollowUpOn: row.NextFollowUpOn,
	}
	if row.Promise != nil {
		promisedOn, amount := row.Promise.PromisedOn, row.Promise.AmountMinor
		before.PromisedOn = &promisedOn
		before.PromisedAmountMinor = &amount
	}
	return &collectionports.AccountChangePlan{Row: row, Before: before, After: before}, nil
}

// validateFollowUp refuses a follow-up scheduled for a day already over.
func (s *Service) validateFollowUp(
	multiErr *errortypes.MultiError,
	field string,
	followUpOn *int64,
) {
	if followUpOn != nil && *followUpOn+secondsPerDay <= s.now() {
		multiErr.Add(field, errortypes.ErrInvalid, "Schedule the follow-up for today or later")
	}
}

// billingCurrency confirms the customer exists in the tenant and returns the
// currency its promises are recorded in.
func (s *Service) billingCurrency(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (string, error) {
	cus, err := s.customers.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:                    customerID,
		TenantInfo:            tenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{IncludeBillingProfile: true},
	})
	if err != nil {
		return "", err
	}
	if cus.BillingProfile != nil && cus.BillingProfile.BillingCurrency != "" {
		return cus.BillingProfile.BillingCurrency, nil
	}
	return money.DefaultCurrencyCode, nil
}

func (s *Service) publish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
	actor *permissionports.RequestActor,
) {
	if s.realtime == nil {
		return
	}
	auditActor := actor.AuditActor()
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       realtimeResource,
		Action:         "updated",
		RecordID:       customerID,
	}); err != nil {
		s.l.Warn("failed to publish collections change", zap.Error(err))
	}
}

func (s *Service) tellCollector(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	collectorID pulid.ID,
	row *collectionports.WorklistRow,
	currency string,
) {
	if s.notifier == nil || row == nil {
		return
	}
	buID := tenantInfo.BuID
	userID := collectorID
	if _, err := s.notifier.Create(ctx, &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		TargetUserID:   &userID,
		Channel:        notification.ChannelUser,
		EventType:      eventAccountAssigned,
		Priority:       notification.PriorityMedium,
		Title:          "You're collecting from " + row.CustomerName,
		Message: money.FormatMinor(row.OverdueMinor, currency) +
			" is overdue across " + pluralInvoices(row.OpenInvoiceCount) + ".",
		Source: "collectionservice",
		Data: map[string]any{
			"link":       "/accounting/ar/collections?customerId=" + row.CustomerID.String(),
			"customerId": row.CustomerID.String(),
		},
		RelatedEntities: map[string]any{"customerId": row.CustomerID.String()},
	}); err != nil {
		s.l.Warn("failed to tell a collector about an assignment", zap.Error(err))
	}
}

// rank settles the facts the list is ordered by: how the standing promise is
// going, whether the follow-up has come round, and the account's priority.
func rank(row *collectionports.WorklistRow, now int64) {
	if row.Promise != nil {
		row.Promise.Status = collection.PromiseStatusOf(
			row.Promise.AmountMinor, row.Promise.PaidSinceMinor, row.Promise.PromisedOn, now,
		)
	}
	row.FollowUpDue = collection.FollowUpIsDue(row.NextFollowUpOn, now)
	facts := collection.PriorityFacts{
		OldestDaysPastDue: row.OldestDaysPastDue,
		OverdueMinor:      row.OverdueMinor,
		FollowUpDue:       row.FollowUpDue,
		FollowUpScheduled: row.NextFollowUpOn != nil,
	}
	if row.Promise != nil {
		facts.HasPromise = true
		facts.Promise = row.Promise.Status
	}
	row.Priority = collection.PriorityOf(facts)
}

// isDue keeps the accounts that need someone today: a follow-up that has come
// round, a broken promise, or an overdue account nobody has scheduled.
func isDue(row *collectionports.WorklistRow) bool {
	switch {
	case row.FollowUpDue:
		return true
	case row.Promise != nil && row.Promise.Status == collection.PromiseBroken:
		return true
	case row.NextFollowUpOn == nil && row.OverdueMinor > 0:
		return row.Promise == nil || row.Promise.Status != collection.PromiseOpen
	default:
		return false
	}
}

func currencyTotals(byCurrency map[string]int64) []collectionports.CurrencyTotal {
	totals := make([]collectionports.CurrencyTotal, 0, len(byCurrency))
	for code, amount := range byCurrency {
		totals = append(
			totals,
			collectionports.CurrencyTotal{CurrencyCode: code, AmountMinor: amount},
		)
	}
	slices.SortFunc(totals, func(a, b collectionports.CurrencyTotal) int {
		return cmp.Or(
			cmp.Compare(b.AmountMinor, a.AmountMinor),
			strings.Compare(a.CurrencyCode, b.CurrencyCode),
		)
	})
	return totals
}

func clampLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultWorklistLimit
	case limit > MaxWorklistLimit:
		return MaxWorklistLimit
	default:
		return limit
	}
}

func accountRequest(
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) collectionports.GetAccountRequest {
	return collectionports.GetAccountRequest{TenantInfo: tenantInfo, CustomerID: customerID}
}

func actorUserID(actor *permissionports.RequestActor, tenantInfo pagination.TenantInfo) pulid.ID {
	if actor != nil && actor.UserID.IsNotNil() {
		return actor.UserID
	}
	return tenantInfo.UserID
}

func pluralInvoices(count int) string {
	if count == 1 {
		return "1 open invoice"
	}
	return strconv.Itoa(count) + " open invoices"
}
