package journalcardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/journalentryservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalposting"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalreversalports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalsourceports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/manualjournalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	journalEntrySubject  = "journal entry"
	manualJournalSubject = "manual journal"
)

type Params struct {
	fx.In

	Logger    *zap.Logger
	Entries   journalentryports.JournalEntryRepository
	Journals  manualjournalports.ManualJournalRepository
	Sources   journalsourceports.JournalSourceRepository
	Reversals journalreversalports.JournalReversalRepository
	Accounts  glaccountports.GLAccountRepository
	Periods   fiscalperiodports.FiscalPeriodRepository
	Controls  tenantports.AccountingControlRepository
	Users     tenantports.UserRepository
	Customers customerports.CustomerRepository
	Locations locationports.LocationRepository
	Ledger    *journalentryservice.Service
}

type Service struct {
	l         *zap.Logger
	entries   journalentryports.JournalEntryRepository
	journals  manualjournalports.ManualJournalRepository
	sources   journalsourceports.JournalSourceRepository
	reversals journalreversalports.JournalReversalRepository
	accounts  glaccountports.GLAccountRepository
	periods   fiscalperiodports.FiscalPeriodRepository
	controls  tenantports.AccountingControlRepository
	users     tenantports.UserRepository
	customers customerports.CustomerRepository
	locations locationports.LocationRepository
	ledger    *journalentryservice.Service
}

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:         p.Logger.Named("service.journal-card"),
		entries:   p.Entries,
		journals:  p.Journals,
		sources:   p.Sources,
		reversals: p.Reversals,
		accounts:  p.Accounts,
		periods:   p.Periods,
		controls:  p.Controls,
		users:     p.Users,
		customers: p.Customers,
		locations: p.Locations,
		ledger:    p.Ledger,
	}
}

type Line struct {
	LineNumber   int
	GLAccountID  pulid.ID
	AccountCode  string
	AccountName  string
	Category     accounttype.Category
	Description  string
	DebitAmount  int64
	CreditAmount int64
	CustomerID   pulid.ID
	CustomerName string
	LocationID   pulid.ID
	LocationName string
}

type Posting struct {
	Period         *fiscalperiod.FiscalPeriod
	AccountingDate int64
	Refusal        string
}

type Parties struct {
	IncludeCustomers bool
	IncludeLocations bool
}

type GetJournalEntryCardRequest struct {
	TenantInfo       pagination.TenantInfo
	EntryID          pulid.ID
	Parties          Parties
	IncludeReversals bool
}

type JournalEntryCard struct {
	Entry               *journalentry.JournalEntry
	Lines               []*Line
	Period              *fiscalperiod.FiscalPeriod
	Posting             *Posting
	Source              *journalsource.Source
	CreatedBy           *tenant.User
	ApprovedBy          *tenant.User
	PostedBy            *tenant.User
	RejectedBy          *tenant.User
	ReversalOf          *journalentry.JournalEntry
	ReversedBy          *journalentry.JournalEntry
	Reversal            *journalreversal.Reversal
	ReversalRequestedBy *tenant.User
}

type GetManualJournalCardRequest struct {
	TenantInfo     pagination.TenantInfo
	JournalID      pulid.ID
	Parties        Parties
	IncludeEntries bool
}

type ManualJournalCard struct {
	Journal         *manualjournal.Request
	Lines           []*Line
	RequestedPeriod *fiscalperiod.FiscalPeriod
	Posting         *Posting
	CreatedBy       *tenant.User
	ApprovedBy      *tenant.User
	RejectedBy      *tenant.User
	CancelledBy     *tenant.User
	PostedBy        *tenant.User
	PostedAt        *int64
	Entries         []*journalentry.JournalEntry
}

type lineSource struct {
	number      int
	accountID   pulid.ID
	description string
	debit       int64
	credit      int64
	customerID  pulid.ID
	locationID  pulid.ID
}

func (s *Service) GetJournalEntryCard(
	ctx context.Context,
	req *GetJournalEntryCardRequest,
) (*JournalEntryCard, error) {
	entry, err := s.entries.GetByID(ctx, journalentryports.GetJournalEntryByIDRequest{
		ID:         req.EntryID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	card := &JournalEntryCard{Entry: entry}
	if card.Lines, err = s.lines(ctx, req.TenantInfo, entryLines(entry), req.Parties); err != nil {
		return nil, err
	}
	if card.Period, err = s.period(ctx, req.TenantInfo, entry.FiscalPeriodID); err != nil {
		return nil, err
	}
	if entryAwaitsPosting(entry) {
		card.Posting, err = s.posting(
			ctx,
			req.TenantInfo,
			entry.AccountingDate,
			journalEntrySubject,
		)
		if err != nil {
			return nil, err
		}
	}
	if card.Source, err = s.source(ctx, req.TenantInfo, entry.ID); err != nil {
		return nil, err
	}
	if card.ReversalOf, err = s.relatedEntry(ctx, req.TenantInfo, entry.ReversalOfID); err != nil {
		return nil, err
	}
	if card.ReversedBy, err = s.relatedEntry(ctx, req.TenantInfo, entry.ReversedByID); err != nil {
		return nil, err
	}
	if req.IncludeReversals {
		reversals, listErr := s.reversals.ListByJournalEntry(
			ctx,
			journalreversalports.ListJournalReversalsByEntryRequest{
				TenantInfo:     req.TenantInfo,
				JournalEntryID: entry.ID,
			},
		)
		if listErr != nil {
			return nil, listErr
		}
		card.Reversal = PickReversal(entry, reversals)
	}

	ids := []pulid.ID{entry.CreatedByID, entry.ApprovedByID, entry.PostedByID, entry.RejectedByID}
	if card.Reversal != nil {
		ids = append(ids, card.Reversal.RequestedByID)
	}
	users, err := s.people(ctx, req.TenantInfo, ids)
	if err != nil {
		return nil, err
	}
	card.CreatedBy = users[entry.CreatedByID]
	card.ApprovedBy = users[entry.ApprovedByID]
	card.PostedBy = users[entry.PostedByID]
	card.RejectedBy = users[entry.RejectedByID]
	if card.Reversal != nil {
		card.ReversalRequestedBy = users[card.Reversal.RequestedByID]
	}

	return card, nil
}

func (s *Service) GetManualJournalCard(
	ctx context.Context,
	req *GetManualJournalCardRequest,
) (*ManualJournalCard, error) {
	journal, err := s.journals.GetByID(ctx, manualjournalports.GetManualJournalByIDRequest{
		ID:         req.JournalID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	card := &ManualJournalCard{Journal: journal, Entries: []*journalentry.JournalEntry{}}
	if card.Lines, err = s.lines(
		ctx,
		req.TenantInfo,
		journalLines(journal),
		req.Parties,
	); err != nil {
		return nil, err
	}
	if card.RequestedPeriod, err = s.period(
		ctx,
		req.TenantInfo,
		journal.RequestedFiscalPeriodID,
	); err != nil {
		return nil, err
	}
	if journalAwaitsPosting(journal) {
		card.Posting, err = s.posting(
			ctx,
			req.TenantInfo,
			journal.AccountingDate,
			manualJournalSubject,
		)
		if err != nil {
			return nil, err
		}
	}

	var postedByID pulid.ID
	if journal.Status == manualjournal.StatusPosted {
		entries, listErr := s.ledger.ListEntriesBySource(
			ctx,
			req.TenantInfo,
			journalsource.ObjectManualJournalRequest,
			journal.ID.String(),
		)
		if listErr != nil {
			return nil, listErr
		}
		if posted := firstPosted(entries); posted != nil {
			postedByID = posted.PostedByID
			card.PostedAt = posted.PostedAt
		}
		if req.IncludeEntries {
			card.Entries = entries
		}
	}

	users, err := s.people(ctx, req.TenantInfo, []pulid.ID{
		journal.CreatedByID,
		journal.ApprovedByID,
		journal.RejectedByID,
		journal.CancelledByID,
		postedByID,
	})
	if err != nil {
		return nil, err
	}
	card.CreatedBy = users[journal.CreatedByID]
	card.ApprovedBy = users[journal.ApprovedByID]
	card.RejectedBy = users[journal.RejectedByID]
	card.CancelledBy = users[journal.CancelledByID]
	card.PostedBy = users[postedByID]

	return card, nil
}

func PickReversal(
	entry *journalentry.JournalEntry,
	reversals []*journalreversal.Reversal,
) *journalreversal.Reversal {
	if entry.IsReversal {
		for _, reversal := range reversals {
			if reversal != nil && reversal.ReversalJournalEntryID == entry.ID {
				return reversal
			}
		}
		return nil
	}

	var latest *journalreversal.Reversal
	for _, reversal := range reversals {
		if reversal == nil || reversal.OriginalJournalEntryID != entry.ID {
			continue
		}
		if reversalStands(reversal.Status) {
			return reversal
		}
		if latest == nil {
			latest = reversal
		}
	}
	return latest
}

func reversalStands(status journalreversal.Status) bool {
	switch status {
	case journalreversal.StatusRequested,
		journalreversal.StatusPendingApproval,
		journalreversal.StatusApproved,
		journalreversal.StatusPosted:
		return true
	case journalreversal.StatusRejected, journalreversal.StatusCancelled:
		return false
	default:
		return false
	}
}

func entryAwaitsPosting(entry *journalentry.JournalEntry) bool {
	if entry.IsPosted {
		return false
	}
	switch entry.Status {
	case journalentry.StatusDraft, journalentry.StatusPending, journalentry.StatusApproved:
		return true
	case journalentry.StatusPosted,
		journalentry.StatusReversed,
		journalentry.StatusRejected,
		journalentry.StatusVoid:
		return false
	default:
		return false
	}
}

func journalAwaitsPosting(journal *manualjournal.Request) bool {
	switch journal.Status {
	case manualjournal.StatusDraft,
		manualjournal.StatusPendingApproval,
		manualjournal.StatusApproved:
		return true
	case manualjournal.StatusRejected, manualjournal.StatusCancelled, manualjournal.StatusPosted:
		return false
	default:
		return false
	}
}

func firstPosted(entries []*journalentry.JournalEntry) *journalentry.JournalEntry {
	for _, entry := range entries {
		if entry != nil && entry.IsPosted && !entry.IsReversal {
			return entry
		}
	}
	return nil
}

func entryLines(entry *journalentry.JournalEntry) []lineSource {
	out := make([]lineSource, 0, len(entry.Lines))
	for _, line := range entry.Lines {
		if line == nil {
			continue
		}
		out = append(out, lineSource{
			number:      int(line.LineNumber),
			accountID:   line.GLAccountID,
			description: line.Description,
			debit:       line.DebitAmount,
			credit:      line.CreditAmount,
			customerID:  line.CustomerID,
			locationID:  line.LocationID,
		})
	}
	return out
}

func journalLines(journal *manualjournal.Request) []lineSource {
	out := make([]lineSource, 0, len(journal.Lines))
	for _, line := range journal.Lines {
		if line == nil {
			continue
		}
		out = append(out, lineSource{
			number:      line.LineNumber,
			accountID:   line.GLAccountID,
			description: line.Description,
			debit:       line.DebitAmount,
			credit:      line.CreditAmount,
			customerID:  line.CustomerID,
			locationID:  line.LocationID,
		})
	}
	return out
}

func (s *Service) lines(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sources []lineSource,
	parties Parties,
) ([]*Line, error) {
	lines := make([]*Line, 0, len(sources))
	if len(sources) == 0 {
		return lines, nil
	}

	accountIDs := make([]pulid.ID, 0, len(sources))
	customerIDs := make([]pulid.ID, 0, len(sources))
	locationIDs := make([]pulid.ID, 0, len(sources))
	for i := range sources {
		accountIDs = appendUnique(accountIDs, sources[i].accountID)
		if parties.IncludeCustomers {
			customerIDs = appendUnique(customerIDs, sources[i].customerID)
		}
		if parties.IncludeLocations {
			locationIDs = appendUnique(locationIDs, sources[i].locationID)
		}
	}

	accounts, err := s.accountViews(ctx, tenantInfo, accountIDs)
	if err != nil {
		return nil, err
	}
	customers, err := s.customerNames(ctx, tenantInfo, customerIDs)
	if err != nil {
		return nil, err
	}
	locations, err := s.locationNames(ctx, tenantInfo, locationIDs)
	if err != nil {
		return nil, err
	}

	for i := range sources {
		src := &sources[i]
		account := accounts[src.accountID]
		line := &Line{
			LineNumber:   src.number,
			GLAccountID:  src.accountID,
			AccountCode:  account.code,
			AccountName:  account.name,
			Category:     account.category,
			Description:  src.description,
			DebitAmount:  src.debit,
			CreditAmount: src.credit,
		}
		if parties.IncludeCustomers {
			line.CustomerID = src.customerID
			line.CustomerName = customers[src.customerID]
		}
		if parties.IncludeLocations {
			line.LocationID = src.locationID
			line.LocationName = locations[src.locationID]
		}
		lines = append(lines, line)
	}

	return lines, nil
}

type accountView struct {
	code     string
	name     string
	category accounttype.Category
}

func (s *Service) accountViews(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]accountView, error) {
	views := make(map[pulid.ID]accountView, len(ids))
	if len(ids) == 0 {
		return views, nil
	}
	found, err := s.accounts.GetByIDs(ctx, glaccountports.GetGLAccountsByIDsRequest{
		TenantInfo:         tenantInfo,
		GLAccountIDs:       ids,
		IncludeAccountType: true,
	})
	if err != nil {
		return nil, err
	}
	for _, account := range found {
		if account == nil {
			continue
		}
		view := accountView{code: account.AccountCode, name: account.Name}
		if account.AccountType != nil {
			view.category = account.AccountType.Category
		}
		views[account.ID] = view
	}
	return views, nil
}

func (s *Service) customerNames(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	names := make(map[pulid.ID]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}
	found, err := s.customers.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: ids,
	})
	if err != nil {
		return nil, err
	}
	for _, customer := range found {
		if customer != nil {
			names[customer.ID] = customer.Name
		}
	}
	return names, nil
}

func (s *Service) locationNames(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	names := make(map[pulid.ID]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}
	found, err := s.locations.GetByIDs(ctx, locationports.GetLocationsByIDsRequest{
		TenantInfo:  tenantInfo,
		LocationIDs: ids,
	})
	if err != nil {
		return nil, err
	}
	for _, location := range found {
		if location != nil {
			names[location.ID] = location.Name
		}
	}
	return names, nil
}

func (s *Service) period(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*fiscalperiod.FiscalPeriod, error) {
	if id.IsNil() {
		return nil, nil //nolint:nilnil // the journal names no period
	}
	period, err := s.periods.GetByID(ctx, fiscalperiodports.GetFiscalPeriodByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // the period is gone; the card draws none
		}
		return nil, err
	}
	return period, nil
}

func (s *Service) posting(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	date int64,
	subject string,
) (*Posting, error) {
	control, err := s.controls.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	resolved, err := journalposting.ResolvePeriod(
		ctx,
		s.periods,
		&journalposting.ResolvePeriodRequest{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Date:           date,
			Policy:         control.ClosedPeriodPostingPolicy,
			Subject:        subject,
		},
	)
	if err != nil {
		if errortypes.IsBusinessError(err) || errortypes.IsError(err) {
			return &Posting{Refusal: errortypes.LocalizedText(ctx, err)}, nil
		}
		return nil, err
	}

	return &Posting{Period: resolved.Period, AccountingDate: resolved.AccountingDate}, nil
}

func (s *Service) source(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entryID pulid.ID,
) (*journalsource.Source, error) {
	source, err := s.sources.GetByJournalEntry(
		ctx,
		journalsourceports.GetJournalSourceByEntryRequest{
			TenantInfo:     tenantInfo,
			JournalEntryID: entryID,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // an entry booked by hand has no source
		}
		return nil, err
	}
	return source, nil
}

func (s *Service) relatedEntry(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*journalentry.JournalEntry, error) {
	if id.IsNil() {
		return nil, nil //nolint:nilnil // the entry is not part of a reversed pair
	}
	entry, err := s.entries.GetByID(ctx, journalentryports.GetJournalEntryByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // the other entry of the pair is gone
		}
		return nil, err
	}
	return entry, nil
}

func (s *Service) people(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]*tenant.User, error) {
	unique := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		unique = appendUnique(unique, id)
	}
	out := make(map[pulid.ID]*tenant.User, len(unique))
	if len(unique) == 0 {
		return out, nil
	}

	users, err := s.users.GetByIDs(ctx, tenantports.GetUsersByIDsRequest{
		TenantInfo: tenantInfo,
		UserIDs:    unique,
	})
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user != nil {
			out[user.ID] = user
		}
	}
	return out, nil
}

func appendUnique(ids []pulid.ID, id pulid.ID) []pulid.ID {
	if id.IsNil() {
		return ids
	}
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}
