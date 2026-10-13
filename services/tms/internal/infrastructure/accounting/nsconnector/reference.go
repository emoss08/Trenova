package nsconnector

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/netsuite"
)

var (
	_ accountingsyncports.AccountingReferenceReader  = (*Connector)(nil)
	_ accountingsyncports.AccountingReferenceCreator = (*Connector)(nil)
)

const (
	maxItemNameLength   = 60
	maxPartyNameLength  = 83
	maxEmailLength      = 254
	maxAddressLength    = 150
	maxCityLength       = 50
	maxStateLength      = 30
	maxPostalCodeLength = 36
	countryCodeLength   = 2
	duplicateMatchLimit = 10
	recordServiceItem   = "serviceSaleItem"
	recordNonInvItem    = "nonInventorySaleItem"
	sublistAddressBook  = "addressBook"
)

var (
	errItemDraftRequired  = errors.New("netsuite: an item draft is required")
	errPartyDraftRequired = errors.New("netsuite: a customer or vendor draft is required")
	errNameRequired       = errors.New("netsuite: a name is required")
	errNoPaymentMethods   = errors.New("netsuite: payment methods are not read from NetSuite")
)

type partyKind int

const (
	partyCustomer partyKind = iota
	partyVendor
)

func (k partyKind) recordType() string {
	if k == partyVendor {
		return recordVendor
	}
	return recordCustomer
}

func (k partyKind) spec() *referenceSpec {
	if k == partyVendor {
		return &vendorSpec
	}
	return &customerSpec
}

func partyKindOf(kind accountingsync.ReferenceKind) partyKind {
	if kind == accountingsync.ReferenceKindVendor {
		return partyVendor
	}
	return partyCustomer
}

func referenceSpecOf(kind accountingsync.ReferenceKind) (*referenceSpec, error) {
	switch kind {
	case accountingsync.ReferenceKindAccount:
		return &accountSpec, nil
	case accountingsync.ReferenceKindItem:
		return &itemSpec, nil
	case accountingsync.ReferenceKindCustomer:
		return &customerSpec, nil
	case accountingsync.ReferenceKindVendor:
		return &vendorSpec, nil
	case accountingsync.ReferenceKindTerm:
		return &termSpec, nil
	case accountingsync.ReferenceKindPaymentMethod:
		return nil, errNoPaymentMethods
	default:
		return nil, fmt.Errorf("netsuite: unknown reference kind %q", kind)
	}
}

func (c *Connector) MaxReferencePageSize() int {
	return netsuite.MaxQueryLimit
}

func (c *Connector) ListReference(
	ctx context.Context,
	req *accountingsyncports.AccountingReferencePageRequest,
) (*accountingsyncports.AccountingReferencePage, error) {
	if req.Kind == accountingsync.ReferenceKindPaymentMethod {
		return &accountingsyncports.AccountingReferencePage{
			Objects: []*accountingsync.AccountingReferenceObject{},
		}, nil
	}
	spec, err := referenceSpecOf(req.Kind)
	if err != nil {
		return nil, err
	}
	s, err := c.open(accountingsyncports.AccountingDocumentAuth{
		RealmID:     req.RealmID,
		AccessToken: req.AccessToken,
	})
	if err != nil {
		return nil, err
	}
	limit := req.PageSize
	if limit <= 0 || limit > netsuite.MaxQueryLimit {
		limit = netsuite.MaxQueryLimit
	}
	b := (&sqlBuilder{}).
		where(spec.filter).
		where(spec.column("id")+" >= ?", max(req.StartPosition, 1))
	if s.ref.SubsidiaryID != "" {
		b.where(spec.subsidiary, s.ref.SubsidiaryID)
	}
	var rows []referenceRow
	page, err := s.client.Query(ctx, netsuite.Query{
		SQL:    b.sql(spec.selectFrom, spec.column("id")),
		Params: b.params,
		Limit:  limit,
	}, &rows)
	if err != nil {
		return nil, err
	}
	out := &accountingsyncports.AccountingReferencePage{
		Objects: make([]*accountingsync.AccountingReferenceObject, 0, len(rows)),
	}
	for idx := range rows {
		out.Objects = append(out.Objects, referenceObjectOf(req.Kind, &rows[idx]))
	}
	if page.HasMore && len(rows) > 0 {
		if last, ok := parseInternalID(rows[len(rows)-1].ID.Value()); ok {
			out.NextStart = int(last + 1)
		}
	}
	return out, nil
}

func referenceObjectOf(
	kind accountingsync.ReferenceKind,
	row *referenceRow,
) *accountingsync.AccountingReferenceObject {
	if kind == accountingsync.ReferenceKindAccount {
		return accountObjectOf(row)
	}
	if kind == accountingsync.ReferenceKindItem {
		return itemObjectOf(row)
	}
	if kind == accountingsync.ReferenceKindTerm {
		return termObjectOf(row)
	}
	return partyObjectOf(kind, row)
}

func accountObjectOf(row *referenceRow) *accountingsync.AccountingReferenceObject {
	name := row.AccountName.Value()
	if name == "" {
		name = row.FullName.Value()
	}
	class := AccountClassOf(row.AccountType.Value())
	return &accountingsync.AccountingReferenceObject{
		Kind:               accountingsync.ReferenceKindAccount,
		ExternalID:         row.ID.Value(),
		Name:               name,
		FullyQualifiedName: row.FullName.Value(),
		Number:             row.Number.Value(),
		AccountType:        row.AccountType.Value(),
		AccountClass:       class,
		ParentExternalID:   row.Parent.Value(),
		Active:             !row.Inactive.Value() && !row.Summary.Value() && class != "",
		ProviderUpdatedAt:  unixOf(row.modified()),
	}
}

func AccountClassOf(accountType string) accountingsync.AccountClass {
	switch strings.TrimSpace(accountType) {
	case "Bank":
		return accountingsync.AccountClassBank
	case "AcctRec", "UnbilledRec":
		return accountingsync.AccountClassReceivable
	case "AcctPay":
		return accountingsync.AccountClassPayable
	case "OthCurrAsset", "FixedAsset", "OthAsset", "DeferExpense":
		return accountingsync.AccountClassAsset
	case "CredCard", "OthCurrLiab", "LongTermLiab", "DeferRevenue":
		return accountingsync.AccountClassLiability
	case "Equity":
		return accountingsync.AccountClassEquity
	case "Income":
		return accountingsync.AccountClassIncome
	case "OthIncome":
		return accountingsync.AccountClassOtherIncome
	case "COGS":
		return accountingsync.AccountClassCostOfSales
	case "Expense":
		return accountingsync.AccountClassExpense
	case "OthExpense":
		return accountingsync.AccountClassOtherExpense
	default:
		return ""
	}
}

func itemObjectOf(row *referenceRow) *accountingsync.AccountingReferenceObject {
	name := row.DisplayName.Value()
	if name == "" {
		name = row.ItemID.Value()
	}
	return &accountingsync.AccountingReferenceObject{
		Kind:                    accountingsync.ReferenceKindItem,
		ExternalID:              row.ID.Value(),
		Name:                    name,
		Number:                  row.ItemID.Value(),
		ItemType:                row.ItemType.Value(),
		SubType:                 row.SubType.Value(),
		IncomeAccountExternalID: row.IncomeAccount.Value(),
		Active:                  !row.Inactive.Value(),
		ProviderUpdatedAt:       unixOf(row.modified()),
	}
}

func partyObjectOf(
	kind accountingsync.ReferenceKind,
	row *referenceRow,
) *accountingsync.AccountingReferenceObject {
	name := row.CompanyName.Value()
	if name == "" {
		name = row.EntityID.Value()
	}
	return &accountingsync.AccountingReferenceObject{
		Kind:              kind,
		ExternalID:        row.ID.Value(),
		Name:              name,
		Number:            row.EntityID.Value(),
		CompanyName:       row.CompanyName.Value(),
		Email:             row.Email.Value(),
		CurrencyCode:      strings.ToUpper(row.Currency.Value()),
		Is1099:            kind == accountingsync.ReferenceKindVendor && row.Is1099.Value(),
		Active:            !row.Inactive.Value(),
		ProviderUpdatedAt: unixOf(row.modified()),
	}
}

func termObjectOf(row *referenceRow) *accountingsync.AccountingReferenceObject {
	obj := &accountingsync.AccountingReferenceObject{
		Kind:       accountingsync.ReferenceKindTerm,
		ExternalID: row.ID.Value(),
		Name:       row.Name.Value(),
		Active:     !row.Inactive.Value(),
	}
	if row.DueDays != nil && row.DueDays.Value() >= 0 {
		days := int(row.DueDays.Value())
		obj.DueDays = &days
	}
	return obj
}

func (c *Connector) CreateReference(
	ctx context.Context,
	req *accountingsyncports.AccountingCreateReferenceRequest,
) (*accountingsync.AccountingReferenceObject, error) {
	switch req.Kind {
	case accountingsync.ReferenceKindItem:
		if req.Item == nil {
			return nil, errItemDraftRequired
		}
	case accountingsync.ReferenceKindCustomer, accountingsync.ReferenceKindVendor:
		if req.Party == nil {
			return nil, errPartyDraftRequired
		}
	case accountingsync.ReferenceKindAccount,
		accountingsync.ReferenceKindTerm,
		accountingsync.ReferenceKindPaymentMethod:
		return nil, fmt.Errorf("netsuite: %s records are not created from Trenova", req.Kind)
	default:
		return nil, fmt.Errorf("netsuite: unknown reference kind %q", req.Kind)
	}
	s, err := c.open(accountingsyncports.AccountingDocumentAuth{
		RealmID:     req.RealmID,
		AccessToken: req.AccessToken,
	})
	if err != nil {
		return nil, err
	}
	if req.Kind == accountingsync.ReferenceKindItem {
		id, createErr := c.createItem(ctx, s, req.RequestID, req.Item)
		if createErr != nil {
			return nil, createErr
		}
		return readReference(ctx, s, req.Kind, id)
	}
	id, err := createParty(ctx, s, &partyCreate{
		kind:      partyKindOf(req.Kind),
		requestID: req.RequestID,
		draft:     req.Party,
	})
	if err != nil {
		return nil, err
	}
	return readReference(ctx, s, req.Kind, id)
}

func readReference(
	ctx context.Context,
	s *session,
	kind accountingsync.ReferenceKind,
	id string,
) (*accountingsync.AccountingReferenceObject, error) {
	spec, err := referenceSpecOf(kind)
	if err != nil {
		return nil, err
	}
	internalID, ok := parseInternalID(id)
	if !ok {
		return nil, netsuite.ErrInvalidRecordID
	}
	b := (&sqlBuilder{}).where(spec.column("id")+" = ?", internalID)
	var rows []referenceRow
	if _, err = s.client.Query(ctx, netsuite.Query{
		SQL:    b.sql(spec.selectFrom, ""),
		Params: b.params,
		Limit:  1,
	}, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, netsuite.ErrUnexpectedPayload
	}
	return referenceObjectOf(kind, &rows[0]), nil
}

func duplicateOf(
	ctx context.Context,
	s *session,
	b *sqlBuilder,
	spec *referenceSpec,
	requestID string,
) error {
	var rows []referenceRow
	if _, err := s.client.Query(ctx, netsuite.Query{
		SQL:    b.sql(spec.selectFrom, spec.column("id")),
		Params: b.params,
		Limit:  duplicateMatchLimit,
	}, &rows); err != nil {
		return err
	}
	for idx := range rows {
		if requestID == "" || rows[idx].ExternalID.Value() != requestID {
			return errDuplicateName
		}
	}
	return nil
}

type itemBody struct {
	ItemID           string              `json:"itemId"`
	DisplayName      string              `json:"displayName,omitempty"`
	SalesDescription string              `json:"salesDescription,omitempty"`
	IncomeAccount    *recordRef          `json:"incomeAccount,omitempty"`
	Subsidiary       *sublist[recordRef] `json:"subsidiary,omitempty"`
	IncludeChildren  bool                `json:"includeChildren"`
}

func (c *Connector) createItem(
	ctx context.Context,
	s *session,
	requestID string,
	draft *accountingsyncports.AccountingItemDraft,
) (string, error) {
	name := c.SanitizeName(accountingsync.ReferenceKindItem, draft.Name)
	if name == "" {
		return "", errNameRequired
	}
	if strings.TrimSpace(requestID) == "" {
		return "", errRequestID
	}
	check := (&sqlBuilder{}).where("UPPER(i.itemid) = UPPER(?)", name)
	if err := duplicateOf(ctx, s, check, &itemSpec, requestID); err != nil {
		return "", fmt.Errorf("%w: item %q", err, name)
	}
	body := &itemBody{
		ItemID:           name,
		DisplayName:      name,
		SalesDescription: cleanText(draft.Description, maxLineTextLength),
		IncomeAccount:    refOf(draft.IncomeAccountID),
	}
	if sub := s.subsidiary(); sub != nil {
		body.Subsidiary = &sublist[recordRef]{Items: []recordRef{*sub}}
	}
	id, err := s.client.Upsert(ctx, netsuite.RecordWrite{
		Type:       recordServiceItem,
		ExternalID: requestID,
		Body:       body,
	})
	if err == nil || !itemTypeRefused(err) {
		return id, err
	}
	return s.client.Upsert(ctx, netsuite.RecordWrite{
		Type:       recordNonInvItem,
		ExternalID: requestID,
		Body:       body,
	})
}

func itemTypeRefused(err error) bool {
	return netsuite.IsNotFound(err) || errorCode(err) == netsuite.CodeInvalidRequest
}

type addressLine struct {
	DefaultBilling     bool        `json:"defaultBilling"`
	DefaultShipping    bool        `json:"defaultShipping"`
	AddressBookAddress addressBody `json:"addressBookAddress"`
}

type addressBody struct {
	Addr1   string     `json:"addr1,omitempty"`
	City    string     `json:"city,omitempty"`
	State   string     `json:"state,omitempty"`
	Zip     string     `json:"zip,omitempty"`
	Country *recordRef `json:"country,omitempty"`
}

type partyBody struct {
	CompanyName    string                `json:"companyName,omitempty"`
	IsPerson       bool                  `json:"isPerson"`
	Email          string                `json:"email,omitempty"`
	Subsidiary     *recordRef            `json:"subsidiary,omitempty"`
	Is1099Eligible *bool                 `json:"is1099Eligible,omitempty"`
	AddressBook    *sublist[addressLine] `json:"addressBook,omitempty"`
}

func partyBodyOf(kind partyKind, draft *accountingsyncports.AccountingPartyDraft) *partyBody {
	name := cleanText(draft.CompanyName, maxPartyNameLength)
	if name == "" {
		name = cleanText(draft.DisplayName, maxPartyNameLength)
	}
	body := &partyBody{
		CompanyName: name,
		Email:       cleanText(draft.Email, maxEmailLength),
	}
	if kind == partyVendor {
		eligible := draft.Is1099
		body.Is1099Eligible = &eligible
	}
	address := addressBody{
		Addr1: cleanText(draft.AddressLine1, maxAddressLength),
		City:  cleanText(draft.City, maxCityLength),
		State: cleanText(draft.State, maxStateLength),
		Zip:   cleanText(draft.PostalCode, maxPostalCodeLength),
	}
	if country := strings.ToUpper(strings.TrimSpace(draft.Country)); len(country) ==
		countryCodeLength {
		address.Country = &recordRef{ID: country}
	}
	if address != (addressBody{}) {
		body.AddressBook = &sublist[addressLine]{Items: []addressLine{{
			DefaultBilling:     true,
			DefaultShipping:    true,
			AddressBookAddress: address,
		}}}
	}
	return body
}

type partyCreate struct {
	kind      partyKind
	requestID string
	draft     *accountingsyncports.AccountingPartyDraft
}

func createParty(ctx context.Context, s *session, p *partyCreate) (string, error) {
	body := partyBodyOf(p.kind, p.draft)
	if body.CompanyName == "" {
		return "", errNameRequired
	}
	requestID := strings.TrimSpace(p.requestID)
	if requestID == "" {
		return "", errRequestID
	}
	spec := p.kind.spec()
	check := (&sqlBuilder{}).where(
		"(UPPER("+spec.column("entityid")+") = UPPER(?) OR UPPER("+
			spec.column("companyname")+") = UPPER(?))",
		body.CompanyName,
		body.CompanyName,
	)
	if err := duplicateOf(ctx, s, check, spec, requestID); err != nil {
		return "", fmt.Errorf("%w: %q", err, body.CompanyName)
	}
	body.Subsidiary = s.subsidiary()
	return s.client.Upsert(ctx, netsuite.RecordWrite{
		Type:       p.kind.recordType(),
		ExternalID: requestID,
		Body:       body,
	})
}

func (c *Connector) IsDuplicateName(err error) bool {
	return errors.Is(err, errDuplicateName) || netsuite.IsDuplicateExternalID(err)
}

func (c *Connector) SanitizeName(kind accountingsync.ReferenceKind, name string) string {
	if kind == accountingsync.ReferenceKindItem {
		return cleanText(name, maxItemNameLength)
	}
	return cleanText(name, maxPartyNameLength)
}

type partyWrite struct {
	auth       accountingsyncports.AccountingDocumentAuth
	kind       partyKind
	requestID  string
	externalID string
	party      *accountingsyncports.AccountingPartyDraft
}

func (c *Connector) upsertParty(
	ctx context.Context,
	w *partyWrite,
) (*accountingsyncports.AccountingDocumentResult, error) {
	s, err := c.open(w.auth)
	if err != nil {
		return nil, err
	}
	body := partyBodyOf(w.kind, w.party)
	if body.CompanyName == "" {
		return nil, errNameRequired
	}
	id := strings.TrimSpace(w.externalID)
	if id != "" {
		var replace []string
		if body.AddressBook != nil {
			replace = []string{sublistAddressBook}
		}
		if err = s.client.Patch(ctx, netsuite.RecordPatch{
			Type:    w.kind.recordType(),
			ID:      id,
			Body:    body,
			Replace: replace,
		}); err != nil {
			return nil, err
		}
	} else if id, err = createParty(ctx, s, &partyCreate{
		kind:      w.kind,
		requestID: w.requestID,
		draft:     w.party,
	}); err != nil {
		return nil, err
	}
	return &accountingsyncports.AccountingDocumentResult{
		ExternalID: id,
		DocNumber:  body.CompanyName,
		Refs:       map[string]string{},
	}, nil
}
