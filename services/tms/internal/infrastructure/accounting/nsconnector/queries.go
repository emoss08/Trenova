package nsconnector

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/shopspring/decimal"
)

const (
	typeInvoice         = "CustInvc"
	typeCreditMemo      = "CustCred"
	typeCustomerPayment = "CustPymt"
	typeVendorBill      = "VendBill"
	typeVendorCredit    = "VendCred"
	typeVendorPayment   = "VendPymt"

	timeFormat     = "'YYYY-MM-DD HH24:MI:SS'"
	dateFormat     = "'YYYY-MM-DD'"
	maxQueryPages  = 1000
	findMatchLimit = 50
)

const (
	transactionColumns = "SELECT t.id, t.type, t.tranid, t.externalid, t.otherrefnum, t.entity, " +
		"t.foreigntotal, t.foreignamountremaining, t.voided, cur.symbol AS currency, " +
		"TO_CHAR(t.lastmodifieddate, " + timeFormat + ") AS lastmodified"
	transactionFrom = " FROM transaction t LEFT JOIN currency cur ON cur.id = t.currency"
	transactionSQL  = transactionColumns + transactionFrom
	stateSQL        = transactionColumns + ", ap.closed, ap.alllocked, ap.arlocked, ap.aplocked" +
		transactionFrom + " LEFT JOIN accountingperiod ap ON ap.id = t.postingperiod" +
		" WHERE t.id = ?"
	transactionSubsidiary = "EXISTS (SELECT 1 FROM transactionline tl WHERE " +
		"tl.transaction = t.id AND tl.mainline = 'T' AND tl.subsidiary = ?)"

	subsidiariesSQL = "SELECT s.id, s.name, cur.symbol AS currency FROM subsidiary s " +
		"LEFT JOIN currency cur ON cur.id = s.currency " +
		"WHERE s.isinactive = 'F' AND s.iselimination = 'F' AND s.id > ? ORDER BY s.id"
	subsidiarySQL = "SELECT s.id, s.name, s.legalname, s.country, cur.symbol AS currency " +
		"FROM subsidiary s LEFT JOIN currency cur ON cur.id = s.currency WHERE s.id = ?"
	companySQL = "SELECT ci.companyname, ci.legalname, ci.country, cur.symbol AS currency " +
		"FROM companyinformation ci LEFT JOIN currency cur ON cur.id = ci.basecurrency"
	periodsSQL = "SELECT p.id, TO_CHAR(p.startdate, " + dateFormat + ") AS startdate, " +
		"TO_CHAR(p.enddate, " + dateFormat + ") AS enddate, p.isyear, p.isposting, " +
		"p.closed, p.alllocked FROM accountingperiod p " +
		"WHERE (p.isyear = 'T' OR p.isposting = 'T') AND p.id > ? ORDER BY p.id"
	currenciesSQL = "SELECT c.id, c.symbol FROM currency c " +
		"WHERE c.isinactive = 'F' AND c.id > ? ORDER BY c.id"
	deletedSQL = "SELECT dr.recordid AS id, dr.type AS recordtype, " +
		"TO_CHAR(dr.deleteddate, " + timeFormat + ") AS lastmodified FROM deletedrecord dr"
)

type transactionRow struct {
	ID          jsonflex.String `json:"id"`
	Type        jsonflex.String `json:"type"`
	TranID      jsonflex.String `json:"tranid"`
	ExternalID  jsonflex.String `json:"externalid"`
	OtherRefNum jsonflex.String `json:"otherrefnum"`
	Entity      jsonflex.String `json:"entity"`
	Total       decimal.Decimal `json:"foreigntotal"`
	Remaining   decimal.Decimal `json:"foreignamountremaining"`
	Voided      jsonflex.Bool   `json:"voided"`
	Currency    jsonflex.String `json:"currency"`
	Modified    jsonflex.String `json:"lastmodified"`
	Closed      jsonflex.Bool   `json:"closed"`
	AllLocked   jsonflex.Bool   `json:"alllocked"`
	ARLocked    jsonflex.Bool   `json:"arlocked"`
	APLocked    jsonflex.Bool   `json:"aplocked"`
}

func (r *transactionRow) id() string {
	return r.ID.Value()
}

func (r *transactionRow) total() decimal.Decimal {
	return r.Total.Abs()
}

func (r *transactionRow) remaining() decimal.Decimal {
	return r.Remaining.Abs()
}

func (r *transactionRow) applied() bool {
	return r.remaining().LessThan(r.total())
}

func (r *transactionRow) modified() time.Time {
	at, err := netsuite.ParseQueryTime(r.Modified.Value())
	if err != nil {
		return time.Time{}
	}
	return at
}

type referenceRow struct {
	ID            jsonflex.String `json:"id"`
	Number        jsonflex.String `json:"acctnumber"`
	AccountName   jsonflex.String `json:"acctname"`
	FullName      jsonflex.String `json:"fullname"`
	AccountType   jsonflex.String `json:"accttype"`
	Summary       jsonflex.Bool   `json:"issummary"`
	Parent        jsonflex.String `json:"parent"`
	ItemID        jsonflex.String `json:"itemid"`
	DisplayName   jsonflex.String `json:"displayname"`
	ItemType      jsonflex.String `json:"itemtype"`
	SubType       jsonflex.String `json:"subtype"`
	IncomeAccount jsonflex.String `json:"incomeaccount"`
	EntityID      jsonflex.String `json:"entityid"`
	CompanyName   jsonflex.String `json:"companyname"`
	Email         jsonflex.String `json:"email"`
	Currency      jsonflex.String `json:"currency"`
	Is1099        jsonflex.Bool   `json:"is1099eligible"`
	Name          jsonflex.String `json:"name"`
	DueDays       *jsonflex.Int   `json:"daysuntilnetdue"`
	Inactive      jsonflex.Bool   `json:"isinactive"`
	ExternalID    jsonflex.String `json:"externalid"`
	Modified      jsonflex.String `json:"lastmodified"`
}

func (r *referenceRow) modified() time.Time {
	at, err := netsuite.ParseQueryTime(r.Modified.Value())
	if err != nil {
		return time.Time{}
	}
	return at
}

type subsidiaryRow struct {
	ID          jsonflex.String `json:"id"`
	Name        jsonflex.String `json:"name"`
	LegalName   jsonflex.String `json:"legalname"`
	Country     jsonflex.String `json:"country"`
	Currency    jsonflex.String `json:"currency"`
	CompanyName jsonflex.String `json:"companyname"`
}

type periodRow struct {
	ID        jsonflex.String `json:"id"`
	StartDate jsonflex.String `json:"startdate"`
	EndDate   jsonflex.String `json:"enddate"`
	Year      jsonflex.Bool   `json:"isyear"`
	Posting   jsonflex.Bool   `json:"isposting"`
	Closed    jsonflex.Bool   `json:"closed"`
	AllLocked jsonflex.Bool   `json:"alllocked"`
}

type currencyRow struct {
	ID     jsonflex.String `json:"id"`
	Symbol jsonflex.String `json:"symbol"`
}

type deletedRow struct {
	ID         jsonflex.String `json:"id"`
	RecordType jsonflex.String `json:"recordtype"`
	Modified   jsonflex.String `json:"lastmodified"`
}

type referenceSpec struct {
	selectFrom string
	alias      string
	filter     string
	subsidiary string
}

var (
	accountSpec = referenceSpec{
		selectFrom: "SELECT a.id, a.acctnumber, a.acctname, a.fullname, a.accttype, " +
			"a.issummary, a.parent, a.isinactive, " +
			"TO_CHAR(a.lastmodifieddate, " + timeFormat + ") AS lastmodified FROM account a",
		alias: "a",
	}
	itemSpec = referenceSpec{
		selectFrom: "SELECT i.id, i.itemid, i.displayname, i.itemtype, i.subtype, " +
			"i.incomeaccount, i.isinactive, i.externalid, " +
			"TO_CHAR(i.lastmodifieddate, " + timeFormat + ") AS lastmodified FROM item i",
		alias: "i",
		filter: "i.itemtype IN ('Service', 'NonInvtPart', 'OthCharge') " +
			"AND (i.subtype IS NULL OR i.subtype <> 'Purchase')",
		subsidiary: "EXISTS (SELECT 1 FROM itemsubsidiarymap m " +
			"WHERE m.item = i.id AND m.subsidiary = ?)",
	}
	customerSpec = referenceSpec{
		selectFrom: "SELECT c.id, c.entityid, c.companyname, c.email, c.isinactive, " +
			"c.externalid, cur.symbol AS currency, " +
			"TO_CHAR(c.lastmodifieddate, " + timeFormat + ") AS lastmodified " +
			"FROM customer c LEFT JOIN currency cur ON cur.id = c.currency",
		alias: "c",
		subsidiary: "EXISTS (SELECT 1 FROM customersubsidiaryrelationship r " +
			"WHERE r.entity = c.id AND r.subsidiary = ?)",
	}
	vendorSpec = referenceSpec{
		selectFrom: "SELECT v.id, v.entityid, v.companyname, v.email, v.isinactive, " +
			"v.is1099eligible, v.externalid, cur.symbol AS currency, " +
			"TO_CHAR(v.lastmodifieddate, " + timeFormat + ") AS lastmodified " +
			"FROM vendor v LEFT JOIN currency cur ON cur.id = v.currency",
		alias: "v",
		subsidiary: "EXISTS (SELECT 1 FROM vendorsubsidiaryrelationship r " +
			"WHERE r.entity = v.id AND r.subsidiary = ?)",
	}
	termSpec = referenceSpec{
		selectFrom: "SELECT tm.id, tm.name, tm.daysuntilnetdue, tm.isinactive FROM term tm",
		alias:      "tm",
	}
	transactionSpec = referenceSpec{
		selectFrom: transactionSQL,
		alias:      "t",
		subsidiary: transactionSubsidiary,
	}
)

func (s *referenceSpec) column(name string) string {
	return s.alias + "." + name
}

type sqlBuilder struct {
	conditions []string
	params     []any
}

func (b *sqlBuilder) where(condition string, params ...any) *sqlBuilder {
	if condition == "" {
		return b
	}
	b.conditions = append(b.conditions, condition)
	b.params = append(b.params, params...)
	return b
}

func (b *sqlBuilder) sql(selectFrom, order string) string {
	var out strings.Builder
	out.WriteString(selectFrom)
	if len(b.conditions) > 0 {
		out.WriteString(" WHERE ")
		out.WriteString(strings.Join(b.conditions, " AND "))
	}
	if order != "" {
		out.WriteString(" ORDER BY ")
		out.WriteString(order)
	}
	return out.String()
}

type keyset struct {
	at time.Time
	id int64
}

func modifiedKey(column string) string {
	return "TO_CHAR(" + column + ", " + timeFormat + ")"
}

func keysetCondition(key, id string) string {
	return "(" + key + " > ? OR (" + key + " = ? AND " + id + " > ?))"
}

func (k keyset) params() []any {
	stamp := netsuite.FormatQueryTime(k.at)
	return []any{stamp, stamp, k.id}
}

type changeFilter struct {
	spec         *referenceSpec
	pos          keyset
	subsidiaryID string
	typeCode     string
}

func changeQuery(f *changeFilter) *sqlBuilder {
	spec := f.spec
	b := &sqlBuilder{}
	if f.typeCode != "" {
		b.where(spec.column("type")+" = ?", f.typeCode)
	}
	b.where(spec.filter)
	b.where(
		keysetCondition(modifiedKey(spec.column("lastmodifieddate")), spec.column("id")),
		f.pos.params()...,
	)
	subsidiaryID := f.subsidiaryID
	if subsidiaryID != "" && spec.subsidiary != "" {
		b.where(spec.subsidiary, subsidiaryID)
	}
	return b
}

func changeOrder(spec *referenceSpec) string {
	return modifiedKey(spec.column("lastmodifieddate")) + ", " + spec.column("id")
}

func parseInternalID(raw string) (int64, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func queryAll[T any](
	ctx context.Context,
	client *netsuite.Client,
	sql string,
	idOf func(*T) string,
	fixed ...any,
) ([]T, error) {
	out := make([]T, 0, netsuite.MaxQueryLimit)
	var after int64
	for range maxQueryPages {
		var rows []T
		params := make([]any, 0, len(fixed)+1)
		params = append(params, fixed...)
		params = append(params, after)
		page, err := client.Query(ctx, netsuite.Query{SQL: sql, Params: params}, &rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
		if !page.HasMore || len(rows) == 0 {
			return out, nil
		}
		last, ok := parseInternalID(idOf(&rows[len(rows)-1]))
		if !ok || last <= after {
			return out, nil
		}
		after = last
	}
	return out, nil
}

func transactionState(
	ctx context.Context,
	client *netsuite.Client,
	id string,
) (*transactionRow, bool, error) {
	internalID, ok := parseInternalID(id)
	if !ok {
		return nil, false, netsuite.ErrInvalidRecordID
	}
	var rows []transactionRow
	if _, err := client.Query(ctx, netsuite.Query{
		SQL:    stateSQL,
		Params: []any{internalID},
		Limit:  1,
	}, &rows); err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	return &rows[0], true, nil
}
