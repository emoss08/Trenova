package nsrefsync

import (
	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
)

const (
	TimeFormat = "'YYYY-MM-DD HH24:MI:SS'"
)

var (
	AccountSpec = nsbinding.ReferenceSpec{
		SelectFrom: "SELECT a.id, a.acctnumber, a.acctname, a.fullname, a.accttype, " +
			"a.issummary, a.parent, a.isinactive, " +
			"TO_CHAR(a.lastmodifieddate, " + TimeFormat + ") AS lastmodified FROM account a",
		Alias: "a",
	}
	ItemSpec = nsbinding.ReferenceSpec{
		SelectFrom: "SELECT i.id, i.itemid, i.displayname, i.itemtype, i.subtype, " +
			"i.incomeaccount, i.isinactive, i.externalid, " +
			"TO_CHAR(i.lastmodifieddate, " + TimeFormat + ") AS lastmodified FROM item i",
		Alias: "i",
		Filter: "i.itemtype IN ('Service', 'NonInvtPart', 'OthCharge') " +
			"AND (i.subtype IS NULL OR i.subtype <> 'Purchase')",
		Subsidiary: "EXISTS (SELECT 1 FROM itemsubsidiarymap m " +
			"WHERE m.item = i.id AND m.subsidiary = ?)",
	}
	CustomerSpec = nsbinding.ReferenceSpec{
		SelectFrom: "SELECT c.id, c.entityid, c.companyname, c.email, c.isinactive, " +
			"c.externalid, cur.symbol AS currency, " +
			"TO_CHAR(c.lastmodifieddate, " + TimeFormat + ") AS lastmodified " +
			"FROM customer c LEFT JOIN currency cur ON cur.id = c.currency",
		Alias: "c",
		Subsidiary: "EXISTS (SELECT 1 FROM customersubsidiaryrelationship r " +
			"WHERE r.entity = c.id AND r.subsidiary = ?)",
	}
	VendorSpec = nsbinding.ReferenceSpec{
		SelectFrom: "SELECT v.id, v.entityid, v.companyname, v.email, v.isinactive, " +
			"v.is1099eligible, v.externalid, cur.symbol AS currency, " +
			"TO_CHAR(v.lastmodifieddate, " + TimeFormat + ") AS lastmodified " +
			"FROM vendor v LEFT JOIN currency cur ON cur.id = v.currency",
		Alias: "v",
		Subsidiary: "EXISTS (SELECT 1 FROM vendorsubsidiaryrelationship r " +
			"WHERE r.entity = v.id AND r.subsidiary = ?)",
	}
	termSpec = nsbinding.ReferenceSpec{
		SelectFrom: "SELECT tm.id, tm.name, tm.daysuntilnetdue, tm.isinactive FROM term tm",
		Alias:      "tm",
	}
)
