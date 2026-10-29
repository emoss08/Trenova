package nsrefsync

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
)

type References struct {
	*nsbinding.Binding
}

const (
	MaxItemNameLength   = 60
	MaxPartyNameLength  = 83
	maxEmailLength      = 254
	maxAddressLength    = 150
	maxCityLength       = 50
	maxStateLength      = 30
	maxPostalCodeLength = 36
	countryCodeLength   = 2
	duplicateMatchLimit = 10
	RecordServiceItem   = "serviceSaleItem"
	RecordNonInvItem    = "nonInventorySaleItem"
)

var (
	errNoPaymentMethods = errors.New("netsuite: payment methods are not read from NetSuite")
)

type PartyKind int

const (
	PartyCustomer PartyKind = iota
	PartyVendor
)

func (k PartyKind) recordType() string {
	if k == PartyVendor {
		return RecordVendor
	}
	return RecordCustomer
}

func (k PartyKind) spec() *nsbinding.ReferenceSpec {
	if k == PartyVendor {
		return &VendorSpec
	}
	return &CustomerSpec
}

func partyKindOf(kind accountingprovider.ReferenceKind) PartyKind {
	if kind == accountingprovider.ReferenceKindVendor {
		return PartyVendor
	}
	return PartyCustomer
}

func referenceSpecOf(kind accountingprovider.ReferenceKind) (*nsbinding.ReferenceSpec, error) {
	switch kind {
	case accountingprovider.ReferenceKindAccount:
		return &AccountSpec, nil
	case accountingprovider.ReferenceKindItem:
		return &ItemSpec, nil
	case accountingprovider.ReferenceKindCustomer:
		return &CustomerSpec, nil
	case accountingprovider.ReferenceKindVendor:
		return &VendorSpec, nil
	case accountingprovider.ReferenceKindTerm:
		return &termSpec, nil
	case accountingprovider.ReferenceKindPaymentMethod:
		return nil, errNoPaymentMethods
	default:
		return nil, fmt.Errorf("netsuite: unknown reference kind %q", kind)
	}
}

func (c *References) MaxReferencePageSize() int {
	return netsuite.MaxQueryLimit
}

func (c *References) ListReference(
	ctx context.Context,
	req *accountingconnectorports.AccountingReferencePageRequest,
) (*accountingconnectorports.AccountingReferencePage, error) {
	if req.Kind == accountingprovider.ReferenceKindPaymentMethod {
		return &accountingconnectorports.AccountingReferencePage{
			Objects: []*accountingmapping.AccountingReferenceObject{},
		}, nil
	}
	spec, err := referenceSpecOf(req.Kind)
	if err != nil {
		return nil, err
	}
	s, err := c.Open(accountingconnectorports.AccountingDocumentAuth{
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
	b := (&nsbinding.SQLBuilder{}).
		Where(spec.Filter).
		Where(spec.Column("id")+" >= ?", max(req.StartPosition, 1))
	if s.Ref.SubsidiaryID != "" {
		b.Where(spec.Subsidiary, s.Ref.SubsidiaryID)
	}
	var rows []nsbinding.ReferenceRow
	page, err := s.Client.Query(ctx, netsuite.Query{
		SQL:    b.SQL(spec.SelectFrom, spec.Column("id")),
		Params: b.Params,
		Limit:  limit,
	}, &rows)
	if err != nil {
		return nil, err
	}
	out := &accountingconnectorports.AccountingReferencePage{
		Objects: make([]*accountingmapping.AccountingReferenceObject, 0, len(rows)),
	}
	for idx := range rows {
		out.Objects = append(out.Objects, ReferenceObjectOf(req.Kind, &rows[idx]))
	}
	if page.HasMore && len(rows) > 0 {
		if last, ok := nsbinding.ParseInternalID(rows[len(rows)-1].ID.Value()); ok {
			out.NextStart = int(last + 1)
		}
	}
	return out, nil
}

func ReferenceObjectOf(
	kind accountingprovider.ReferenceKind,
	row *nsbinding.ReferenceRow,
) *accountingmapping.AccountingReferenceObject {
	if kind == accountingprovider.ReferenceKindAccount {
		return accountObjectOf(row)
	}
	if kind == accountingprovider.ReferenceKindItem {
		return itemObjectOf(row)
	}
	if kind == accountingprovider.ReferenceKindTerm {
		return termObjectOf(row)
	}
	return partyObjectOf(kind, row)
}

func accountObjectOf(row *nsbinding.ReferenceRow) *accountingmapping.AccountingReferenceObject {
	name := row.AccountName.Value()
	if name == "" {
		name = row.FullName.Value()
	}
	class := AccountClassOf(row.AccountType.Value())
	return &accountingmapping.AccountingReferenceObject{
		Kind:               accountingprovider.ReferenceKindAccount,
		ExternalID:         row.ID.Value(),
		Name:               name,
		FullyQualifiedName: row.FullName.Value(),
		Number:             row.Number.Value(),
		AccountType:        row.AccountType.Value(),
		AccountClass:       class,
		ParentExternalID:   row.Parent.Value(),
		Active:             !row.Inactive.Value() && !row.Summary.Value() && class != "",
		ProviderUpdatedAt:  unixOf(row.ModifiedAt()),
	}
}

func AccountClassOf(accountType string) accountingmapping.AccountClass {
	switch strings.TrimSpace(accountType) {
	case "Bank":
		return accountingmapping.AccountClassBank
	case "AcctRec", "UnbilledRec":
		return accountingmapping.AccountClassReceivable
	case "AcctPay":
		return accountingmapping.AccountClassPayable
	case "OthCurrAsset", "FixedAsset", "OthAsset", "DeferExpense":
		return accountingmapping.AccountClassAsset
	case "CredCard", "OthCurrLiab", "LongTermLiab", "DeferRevenue":
		return accountingmapping.AccountClassLiability
	case "Equity":
		return accountingmapping.AccountClassEquity
	case "Income":
		return accountingmapping.AccountClassIncome
	case "OthIncome":
		return accountingmapping.AccountClassOtherIncome
	case "COGS":
		return accountingmapping.AccountClassCostOfSales
	case "Expense":
		return accountingmapping.AccountClassExpense
	case "OthExpense":
		return accountingmapping.AccountClassOtherExpense
	default:
		return ""
	}
}

func itemObjectOf(row *nsbinding.ReferenceRow) *accountingmapping.AccountingReferenceObject {
	name := row.DisplayName.Value()
	if name == "" {
		name = row.ItemID.Value()
	}
	return &accountingmapping.AccountingReferenceObject{
		Kind:                    accountingprovider.ReferenceKindItem,
		ExternalID:              row.ID.Value(),
		Name:                    name,
		Number:                  row.ItemID.Value(),
		ItemType:                row.ItemType.Value(),
		SubType:                 row.SubType.Value(),
		IncomeAccountExternalID: row.IncomeAccount.Value(),
		Active:                  !row.Inactive.Value(),
		ProviderUpdatedAt:       unixOf(row.ModifiedAt()),
	}
}

func partyObjectOf(
	kind accountingprovider.ReferenceKind,
	row *nsbinding.ReferenceRow,
) *accountingmapping.AccountingReferenceObject {
	name := row.CompanyName.Value()
	if name == "" {
		name = row.EntityID.Value()
	}
	return &accountingmapping.AccountingReferenceObject{
		Kind:              kind,
		ExternalID:        row.ID.Value(),
		Name:              name,
		Number:            row.EntityID.Value(),
		CompanyName:       row.CompanyName.Value(),
		Email:             row.Email.Value(),
		CurrencyCode:      strings.ToUpper(row.Currency.Value()),
		Is1099:            kind == accountingprovider.ReferenceKindVendor && row.Is1099.Value(),
		Active:            !row.Inactive.Value(),
		ProviderUpdatedAt: unixOf(row.ModifiedAt()),
	}
}

func termObjectOf(row *nsbinding.ReferenceRow) *accountingmapping.AccountingReferenceObject {
	obj := &accountingmapping.AccountingReferenceObject{
		Kind:       accountingprovider.ReferenceKindTerm,
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

func (c *References) CreateReference(
	ctx context.Context,
	req *accountingconnectorports.AccountingCreateReferenceRequest,
) (*accountingmapping.AccountingReferenceObject, error) {
	switch req.Kind {
	case accountingprovider.ReferenceKindItem:
		if req.Item == nil {
			return nil, nsbinding.ErrItemDraftRequired
		}
	case accountingprovider.ReferenceKindCustomer, accountingprovider.ReferenceKindVendor:
		if req.Party == nil {
			return nil, nsbinding.ErrPartyDraftRequired
		}
	case accountingprovider.ReferenceKindAccount,
		accountingprovider.ReferenceKindTerm,
		accountingprovider.ReferenceKindPaymentMethod:
		return nil, fmt.Errorf("netsuite: %s records are not created from Trenova", req.Kind)
	default:
		return nil, fmt.Errorf("netsuite: unknown reference kind %q", req.Kind)
	}
	s, err := c.Open(accountingconnectorports.AccountingDocumentAuth{
		RealmID:     req.RealmID,
		AccessToken: req.AccessToken,
	})
	if err != nil {
		return nil, err
	}
	if req.Kind == accountingprovider.ReferenceKindItem {
		id, createErr := c.createItem(ctx, s, req.RequestID, req.Item)
		if createErr != nil {
			return nil, createErr
		}
		return readReference(ctx, s, req.Kind, id)
	}
	id, err := CreateParty(ctx, s, &PartyCreate{
		Kind:      partyKindOf(req.Kind),
		RequestID: req.RequestID,
		Draft:     req.Party,
	})
	if err != nil {
		return nil, err
	}
	return readReference(ctx, s, req.Kind, id)
}

func readReference(
	ctx context.Context,
	s *nsbinding.Session,
	kind accountingprovider.ReferenceKind,
	id string,
) (*accountingmapping.AccountingReferenceObject, error) {
	spec, err := referenceSpecOf(kind)
	if err != nil {
		return nil, err
	}
	internalID, ok := nsbinding.ParseInternalID(id)
	if !ok {
		return nil, netsuiteerrors.ErrInvalidRecordID
	}
	b := (&nsbinding.SQLBuilder{}).Where(spec.Column("id")+" = ?", internalID)
	var rows []nsbinding.ReferenceRow
	if _, err = s.Client.Query(ctx, netsuite.Query{
		SQL:    b.SQL(spec.SelectFrom, ""),
		Params: b.Params,
		Limit:  1,
	}, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, netsuiteerrors.ErrUnexpectedPayload
	}
	return ReferenceObjectOf(kind, &rows[0]), nil
}

func duplicateOf(
	ctx context.Context,
	s *nsbinding.Session,
	b *nsbinding.SQLBuilder,
	spec *nsbinding.ReferenceSpec,
	requestID string,
) error {
	var rows []nsbinding.ReferenceRow
	if _, err := s.Client.Query(ctx, netsuite.Query{
		SQL:    b.SQL(spec.SelectFrom, spec.Column("id")),
		Params: b.Params,
		Limit:  duplicateMatchLimit,
	}, &rows); err != nil {
		return err
	}
	for idx := range rows {
		if requestID == "" || rows[idx].ExternalID.Value() != requestID {
			return nsbinding.ErrDuplicateName
		}
	}
	return nil
}

type itemBody struct {
	ItemID           string                                  `json:"itemId"`
	DisplayName      string                                  `json:"displayName,omitempty"`
	SalesDescription string                                  `json:"salesDescription,omitempty"`
	IncomeAccount    *nsbinding.RecordRef                    `json:"incomeAccount,omitempty"`
	Subsidiary       *nsbinding.Sublist[nsbinding.RecordRef] `json:"subsidiary,omitempty"`
	IncludeChildren  bool                                    `json:"includeChildren"`
}

func (c *References) createItem(
	ctx context.Context,
	s *nsbinding.Session,
	requestID string,
	draft *accountingconnectorports.AccountingItemDraft,
) (string, error) {
	name := c.SanitizeName(accountingprovider.ReferenceKindItem, draft.Name)
	if name == "" {
		return "", nsbinding.ErrNameRequired
	}
	if strings.TrimSpace(requestID) == "" {
		return "", nsbinding.ErrRequestID
	}
	check := (&nsbinding.SQLBuilder{}).Where("UPPER(i.itemid) = UPPER(?)", name)
	if err := duplicateOf(ctx, s, check, &ItemSpec, requestID); err != nil {
		return "", fmt.Errorf("%w: item %q", err, name)
	}
	body := &itemBody{
		ItemID:           name,
		DisplayName:      name,
		SalesDescription: CleanText(draft.Description, MaxLineTextLength),
		IncomeAccount:    RefOf(draft.IncomeAccountID),
	}
	if sub := s.Subsidiary(); sub != nil {
		body.Subsidiary = &nsbinding.Sublist[nsbinding.RecordRef]{
			Items: []nsbinding.RecordRef{*sub},
		}
	}
	id, err := s.Client.Upsert(ctx, netsuite.RecordWrite{
		Type:       RecordServiceItem,
		ExternalID: requestID,
		Body:       body,
	})
	if err == nil || !itemTypeRefused(err) {
		return id, err
	}
	return s.Client.Upsert(ctx, netsuite.RecordWrite{
		Type:       RecordNonInvItem,
		ExternalID: requestID,
		Body:       body,
	})
}

func itemTypeRefused(err error) bool {
	return netsuiteerrors.IsNotFound(err) ||
		nsbinding.ErrorCode(err) == netsuiteerrors.CodeInvalidRequest
}

type addressLine struct {
	DefaultBilling     bool        `json:"defaultBilling"`
	DefaultShipping    bool        `json:"defaultShipping"`
	AddressBookAddress addressBody `json:"addressBookAddress"`
}

type addressBody struct {
	Addr1   string               `json:"addr1,omitempty"`
	City    string               `json:"city,omitempty"`
	State   string               `json:"state,omitempty"`
	Zip     string               `json:"zip,omitempty"`
	Country *nsbinding.RecordRef `json:"country,omitempty"`
}

type partyBody struct {
	CompanyName    string                          `json:"companyName,omitempty"`
	IsPerson       bool                            `json:"isPerson"`
	Email          string                          `json:"email,omitempty"`
	Subsidiary     *nsbinding.RecordRef            `json:"subsidiary,omitempty"`
	Is1099Eligible *bool                           `json:"is1099Eligible,omitempty"`
	AddressBook    *nsbinding.Sublist[addressLine] `json:"addressBook,omitempty"`
}

func PartyBodyOf(kind PartyKind, draft *accountingconnectorports.AccountingPartyDraft) *partyBody {
	name := CleanText(draft.CompanyName, MaxPartyNameLength)
	if name == "" {
		name = CleanText(draft.DisplayName, MaxPartyNameLength)
	}
	body := &partyBody{
		CompanyName: name,
		Email:       CleanText(draft.Email, maxEmailLength),
	}
	if kind == PartyVendor {
		eligible := draft.Is1099
		body.Is1099Eligible = &eligible
	}
	address := addressBody{
		Addr1: CleanText(draft.AddressLine1, maxAddressLength),
		City:  CleanText(draft.City, maxCityLength),
		State: CleanText(draft.State, maxStateLength),
		Zip:   CleanText(draft.PostalCode, maxPostalCodeLength),
	}
	if country := strings.ToUpper(strings.TrimSpace(draft.Country)); len(country) ==
		countryCodeLength {
		address.Country = &nsbinding.RecordRef{ID: country}
	}
	if address != (addressBody{}) {
		body.AddressBook = &nsbinding.Sublist[addressLine]{Items: []addressLine{{
			DefaultBilling:     true,
			DefaultShipping:    true,
			AddressBookAddress: address,
		}}}
	}
	return body
}

type PartyCreate struct {
	Kind      PartyKind
	RequestID string
	Draft     *accountingconnectorports.AccountingPartyDraft
}

func CreateParty(ctx context.Context, s *nsbinding.Session, p *PartyCreate) (string, error) {
	body := PartyBodyOf(p.Kind, p.Draft)
	if body.CompanyName == "" {
		return "", nsbinding.ErrNameRequired
	}
	requestID := strings.TrimSpace(p.RequestID)
	if requestID == "" {
		return "", nsbinding.ErrRequestID
	}
	spec := p.Kind.spec()
	check := (&nsbinding.SQLBuilder{}).Where(
		"(UPPER("+spec.Column("entityid")+") = UPPER(?) OR UPPER("+
			spec.Column("companyname")+") = UPPER(?))",
		body.CompanyName,
		body.CompanyName,
	)
	if err := duplicateOf(ctx, s, check, spec, requestID); err != nil {
		return "", fmt.Errorf("%w: %q", err, body.CompanyName)
	}
	body.Subsidiary = s.Subsidiary()
	return s.Client.Upsert(ctx, netsuite.RecordWrite{
		Type:       p.Kind.recordType(),
		ExternalID: requestID,
		Body:       body,
	})
}

func (c *References) IsDuplicateName(err error) bool {
	return errors.Is(err, nsbinding.ErrDuplicateName) || netsuiteerrors.IsDuplicateExternalID(err)
}

func (c *References) SanitizeName(kind accountingprovider.ReferenceKind, name string) string {
	if kind == accountingprovider.ReferenceKindItem {
		return CleanText(name, MaxItemNameLength)
	}
	return CleanText(name, MaxPartyNameLength)
}
