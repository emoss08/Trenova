package nsbinding

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
	maxQueryPages = 1000
)

const (
	subsidiarySQL = "SELECT s.id, s.name, s.legalname, s.country, cur.symbol AS currency " +
		"FROM subsidiary s LEFT JOIN currency cur ON cur.id = s.currency WHERE s.id = ?"
	CompanySQL = "SELECT ci.companyname, ci.legalname, ci.country, cur.symbol AS currency " +
		"FROM companyinformation ci LEFT JOIN currency cur ON cur.id = ci.basecurrency"
	CurrenciesSQL = "SELECT c.id, c.symbol FROM currency c " +
		"WHERE c.isinactive = 'F' AND c.id > ? ORDER BY c.id"
	transactionRecordTypeID = "-30"
	editorsSQL              = "SELECT sn.recordid AS id, BUILTIN.DF(sn.name) AS modifiedby FROM systemnote sn " +
		"WHERE sn.id IN (SELECT MAX(n.id) FROM systemnote n WHERE n.recordtypeid = " +
		transactionRecordTypeID + " AND n.recordid IN ("
)

type TransactionRow struct {
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

func (r *TransactionRow) IDValue() string {
	return r.ID.Value()
}

func (r *TransactionRow) AbsTotal() decimal.Decimal {
	return r.Total.Abs()
}

func (r *TransactionRow) AbsRemaining() decimal.Decimal {
	return r.Remaining.Abs()
}

func (r *TransactionRow) Applied() bool {
	return r.AbsRemaining().LessThan(r.AbsTotal())
}

func (r *TransactionRow) ModifiedAt() time.Time {
	at, err := netsuite.ParseQueryTime(r.Modified.Value())
	if err != nil {
		return time.Time{}
	}
	return at
}

type ReferenceRow struct {
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

func (r *ReferenceRow) ModifiedAt() time.Time {
	at, err := netsuite.ParseQueryTime(r.Modified.Value())
	if err != nil {
		return time.Time{}
	}
	return at
}

type SubsidiaryRow struct {
	ID          jsonflex.String `json:"id"`
	Name        jsonflex.String `json:"name"`
	LegalName   jsonflex.String `json:"legalname"`
	Country     jsonflex.String `json:"country"`
	Currency    jsonflex.String `json:"currency"`
	CompanyName jsonflex.String `json:"companyname"`
}

type CurrencyRow struct {
	ID     jsonflex.String `json:"id"`
	Symbol jsonflex.String `json:"symbol"`
}

type editorRow struct {
	ID         jsonflex.String `json:"id"`
	ModifiedBy jsonflex.String `json:"modifiedby"`
}

type ReferenceSpec struct {
	SelectFrom string
	alias      string
	Filter     string
	Subsidiary string
}

func (s *ReferenceSpec) Column(name string) string {
	return s.alias + "." + name
}

type SQLBuilder struct {
	conditions []string
	Params     []any
}

func (b *SQLBuilder) Where(condition string, params ...any) *SQLBuilder {
	if condition == "" {
		return b
	}
	b.conditions = append(b.conditions, condition)
	b.Params = append(b.Params, params...)
	return b
}

func (b *SQLBuilder) SQL(selectFrom, order string) string {
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

type Keyset struct {
	At time.Time
	ID int64
}

func (k Keyset) Params() []any {
	stamp := netsuite.FormatQueryTime(k.At)
	return []any{stamp, stamp, k.ID}
}

func ParseInternalID(raw string) (int64, bool) {
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

func IDList(ids []int64) (marks string, params []any) {
	params = make([]any, 0, len(ids))
	holders := make([]string, 0, len(ids))
	for _, id := range ids {
		params = append(params, id)
		holders = append(holders, "?")
	}
	return strings.Join(holders, ", "), params
}

type EditorBook struct {
	Client     *netsuite.Client
	unreadable bool
}

func (b *EditorBook) Names(ctx context.Context, ids []int64) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	for start := 0; start < len(ids) && !b.unreadable; start += MaxIDsPerRead {
		marks, params := IDList(ids[start:min(start+MaxIDsPerRead, len(ids))])
		var rows []editorRow
		_, err := b.Client.Query(ctx, netsuite.Query{
			SQL:    editorsSQL + marks + ") GROUP BY n.recordid)",
			Params: params,
		}, &rows)
		if err != nil {
			if !isQueryRefused(err) {
				return nil, err
			}
			b.unreadable = true
			return out, nil
		}
		for idx := range rows {
			if name := strings.TrimSpace(rows[idx].ModifiedBy.Value()); name != "" {
				out[rows[idx].ID.Value()] = name
			}
		}
	}
	return out, nil
}

func QueryAll[T any](
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
		last, ok := ParseInternalID(idOf(&rows[len(rows)-1]))
		if !ok || last <= after {
			return out, nil
		}
		after = last
	}
	return out, nil
}
