package nsconnector

import (
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsdocsync"
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsrefsync"
	"github.com/emoss08/trenova/shared/jsonflex"
)

const (
	dateFormat            = "'YYYY-MM-DD'"
	transactionSubsidiary = "EXISTS (SELECT 1 FROM transactionline tl WHERE " +
		"tl.transaction = t.id AND tl.mainline = 'T' AND tl.subsidiary = ?)"
	subsidiariesSQL = "SELECT s.id, s.name, cur.symbol AS currency FROM subsidiary s " +
		"LEFT JOIN currency cur ON cur.id = s.currency " +
		"WHERE s.isinactive = 'F' AND s.iselimination = 'F' AND s.id > ? ORDER BY s.id"
	periodsSQL = "SELECT p.id, TO_CHAR(p.startdate, " + dateFormat + ") AS startdate, " +
		"TO_CHAR(p.enddate, " + dateFormat + ") AS enddate, p.isyear, p.isposting, " +
		"p.closed, p.alllocked FROM accountingperiod p " +
		"WHERE (p.isyear = 'T' OR p.isposting = 'T') AND p.id > ? ORDER BY p.id"
	deletedSQL = "SELECT dr.recordid AS id, dr.type AS recordtype, " +
		"TO_CHAR(dr.deleteddate, " + nsrefsync.TimeFormat + ") AS lastmodified FROM deletedrecord dr"
)

var (
	transactionSpec = nsbinding.ReferenceSpec{
		SelectFrom: nsdocsync.TransactionSQL,
		alias:      "t",
		Subsidiary: transactionSubsidiary,
	}
)

type periodRow struct {
	ID        jsonflex.String `json:"id"`
	StartDate jsonflex.String `json:"startdate"`
	EndDate   jsonflex.String `json:"enddate"`
	Year      jsonflex.Bool   `json:"isyear"`
	Posting   jsonflex.Bool   `json:"isposting"`
	Closed    jsonflex.Bool   `json:"closed"`
	AllLocked jsonflex.Bool   `json:"alllocked"`
}

type deletedRow struct {
	ID         jsonflex.String `json:"id"`
	RecordType jsonflex.String `json:"recordtype"`
	Modified   jsonflex.String `json:"lastmodified"`
}

func modifiedKey(column string) string {
	return "TO_CHAR(" + column + ", " + nsrefsync.TimeFormat + ")"
}

func keysetCondition(key, id string) string {
	return "(" + key + " > ? OR (" + key + " = ? AND " + id + " > ?))"
}

type changeFilter struct {
	spec         *nsbinding.ReferenceSpec
	pos          nsbinding.Keyset
	subsidiaryID string
	typeCode     string
}

func changeQuery(f *changeFilter) *nsbinding.SQLBuilder {
	spec := f.spec
	b := &nsbinding.SQLBuilder{}
	if f.typeCode != "" {
		b.Where(spec.Column("type")+" = ?", f.typeCode)
	}
	b.Where(spec.Filter)
	b.Where(
		keysetCondition(modifiedKey(spec.Column("lastmodifieddate")), spec.Column("id")),
		f.pos.Params()...,
	)
	subsidiaryID := f.subsidiaryID
	if subsidiaryID != "" && spec.Subsidiary != "" {
		b.Where(spec.Subsidiary, subsidiaryID)
	}
	return b
}

func changeOrder(spec *nsbinding.ReferenceSpec) string {
	return modifiedKey(spec.Column("lastmodifieddate")) + ", " + spec.Column("id")
}
