package inboundmessagerepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen/carriercol"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/documenttypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
)

// maxPartyCandidates bounds the rows a sender lookup reads. Recipient lists are
// free text, so the database narrows by substring and the finder confirms the
// address is a whole entry; a handful of rows is plenty for that.
const maxPartyCandidates = 25

// RecordFinder answers the inbox's two questions of the rest of the system:
// what a reference or a sender's address names, and whether a record a message
// is linked to exists in the tenant at all.
//
// Every answer is unique or nothing. An address two customers both list, or a
// reference two shipments share, is exactly the message a person should read.
type RecordFinder struct {
	db *postgres.Connection
}

func NewRecordFinder(p Params) *RecordFinder {
	return &RecordFinder{db: p.DB}
}

func (f *RecordFinder) FindByReference(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	reference string,
) (pulid.ID, bool, error) {
	return dbtx.Read2(ctx, f.db, func(ctx context.Context) (pulid.ID, bool, error) {
		reference = strings.TrimSpace(reference)
		if reference == "" {
			return pulid.Nil, false, nil
		}

		cols := shipmentcol.ShipmentColumns
		forms := bun.In(referenceForms(reference))

		ids := make([]pulid.ID, 0, 2)
		err := f.db.DBForContext(ctx).
			NewSelect().
			Model((*shipment.Shipment)(nil)).
			ColumnExpr(cols.ID.Qualified()).
			Apply(shipmentcol.ShipmentApplyTenant(tenantInfo)).
			WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.
					Where(cols.ProNumber.In(), forms).
					WhereOr(cols.BOL.In(), forms)
			}).
			Limit(2).
			Scan(ctx, &ids)
		if err != nil {
			return pulid.Nil, false, err
		}

		return onlyOne(ids)
	})
}

// referenceForms is the reference as written and upper-cased. Pro numbers are
// stored as the sequence wrote them, which is upper case, and people type them
// however they like; comparing both forms keeps the index usable where a
// LOWER() on the column would not.
func referenceForms(reference string) []string {
	upper := strings.ToUpper(reference)
	if upper == reference {
		return []string{reference}
	}

	return []string{reference, upper}
}

func (f *RecordFinder) FindCustomerByEmail(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	address string,
) (pulid.ID, bool, error) {
	return dbtx.Read2(ctx, f.db, func(ctx context.Context) (pulid.ID, bool, error) {
		address = stringutils.NormalizeEmailAddress(address)
		if address == "" {
			return pulid.Nil, false, nil
		}
		pattern := "%" + stringutils.EscapeLikePattern(address) + "%"
		found := make(map[pulid.ID]struct{}, 2)

		type customerRecipients struct {
			ID         pulid.ID `bun:"id"`
			Recipients string   `bun:"recipients"`
		}

		customerCols := customercol.CustomerColumns
		var customers []customerRecipients
		if err := f.db.DBForContext(ctx).
			NewSelect().
			Model((*customer.Customer)(nil)).
			ColumnExpr(customerCols.ID.As("id")).
			ColumnExpr(customerCols.StatusUpdateRecipients.As("recipients")).
			Apply(customercol.CustomerApplyTenant(tenantInfo)).
			Where(customerCols.StatusUpdateRecipients.LowerLike(), pattern).
			Limit(maxPartyCandidates).
			Scan(ctx, &customers); err != nil {
			return pulid.Nil, false, err
		}
		for _, row := range customers {
			if listsAddress(row.Recipients, address) {
				found[row.ID] = struct{}{}
			}
		}

		type profileRecipients struct {
			CustomerID pulid.ID `bun:"customer_id"`
			To         string   `bun:"to_recipients"`
			CC         string   `bun:"cc_recipients"`
		}

		profileCols := customercol.CustomerEmailProfileColumns
		var profiles []profileRecipients
		if err := f.db.DBForContext(ctx).
			NewSelect().
			Model((*customer.CustomerEmailProfile)(nil)).
			ColumnExpr(profileCols.CustomerID.As("customer_id")).
			ColumnExpr(profileCols.ToRecipients.As("to_recipients")).
			ColumnExpr(profileCols.CCRecipients.As("cc_recipients")).
			Apply(customercol.CustomerEmailProfileApplyTenant(tenantInfo)).
			WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.
					Where(profileCols.ToRecipients.LowerLike(), pattern).
					WhereOr(profileCols.CCRecipients.LowerLike(), pattern)
			}).
			Limit(maxPartyCandidates).
			Scan(ctx, &profiles); err != nil {
			return pulid.Nil, false, err
		}
		for _, row := range profiles {
			if listsAddress(row.To, address) || listsAddress(row.CC, address) {
				found[row.CustomerID] = struct{}{}
			}
		}

		return onlyOneOf(found)
	})
}

// listsAddress confirms the address is a whole entry in a recipient list, so
// "ops@acme.com" does not match "cops@acme.com" or "ops@acme.com.au".
func listsAddress(list, address string) bool {
	for _, entry := range stringutils.SplitEmailList(list) {
		if entry == address {
			return true
		}
	}

	return false
}

func (f *RecordFinder) FindCarrierByEmail(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	address string,
) (pulid.ID, bool, error) {
	return dbtx.Read2(ctx, f.db, func(ctx context.Context) (pulid.ID, bool, error) {
		address = stringutils.NormalizeEmailAddress(address)
		if address == "" {
			return pulid.Nil, false, nil
		}
		found := make(map[pulid.ID]struct{}, 2)

		carrierCols := carriercol.CarrierColumns
		carriers := make([]pulid.ID, 0, 2)
		if err := f.db.DBForContext(ctx).
			NewSelect().
			Model((*carrier.Carrier)(nil)).
			ColumnExpr(carrierCols.ID.Qualified()).
			Apply(carriercol.CarrierApplyTenant(tenantInfo)).
			Where(carrierCols.Email.LowerLike(), stringutils.EscapeLikePattern(address)).
			Limit(maxPartyCandidates).
			Scan(ctx, &carriers); err != nil {
			return pulid.Nil, false, err
		}
		for _, id := range carriers {
			found[id] = struct{}{}
		}

		contactCols := carriercol.CarrierContactColumns
		contacts := make([]pulid.ID, 0, 2)
		if err := f.db.DBForContext(ctx).
			NewSelect().
			Model((*carrier.CarrierContact)(nil)).
			ColumnExpr(contactCols.CarrierID.Qualified()).
			Apply(carriercol.CarrierContactApplyTenant(tenantInfo)).
			Where(contactCols.Email.LowerLike(), stringutils.EscapeLikePattern(address)).
			Limit(maxPartyCandidates).
			Scan(ctx, &contacts); err != nil {
			return pulid.Nil, false, err
		}
		for _, id := range contacts {
			found[id] = struct{}{}
		}

		return onlyOneOf(found)
	})
}

func (f *RecordFinder) ShipmentExists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, f.db, func(ctx context.Context) (bool, error) {
		return f.db.DBForContext(ctx).
			NewSelect().
			Model((*shipment.Shipment)(nil)).
			Where(shipmentcol.ShipmentColumns.ID.Eq(), id).
			Apply(shipmentcol.ShipmentApplyTenant(tenantInfo)).
			Exists(ctx)
	})
}

func (f *RecordFinder) CustomerExists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, f.db, func(ctx context.Context) (bool, error) {
		return f.db.DBForContext(ctx).
			NewSelect().
			Model((*customer.Customer)(nil)).
			Where(customercol.CustomerColumns.ID.Eq(), id).
			Apply(customercol.CustomerApplyTenant(tenantInfo)).
			Exists(ctx)
	})
}

func (f *RecordFinder) CarrierExists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, f.db, func(ctx context.Context) (bool, error) {
		return f.db.DBForContext(ctx).
			NewSelect().
			Model((*carrier.Carrier)(nil)).
			Where(carriercol.CarrierColumns.ID.Eq(), id).
			Apply(carriercol.CarrierApplyTenant(tenantInfo)).
			Exists(ctx)
	})
}

func onlyOne(ids []pulid.ID) (pulid.ID, bool, error) {
	if len(ids) != 1 {
		return pulid.Nil, false, nil
	}

	return ids[0], true, nil
}

func onlyOneOf(found map[pulid.ID]struct{}) (pulid.ID, bool, error) {
	if len(found) != 1 {
		return pulid.Nil, false, nil
	}
	for id := range found {
		return id, true, nil
	}

	return pulid.Nil, false, nil
}

func (f *RecordFinder) DocumentTypeNames(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, f.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		names := make(map[pulid.ID]string, len(ids))
		if len(ids) == 0 {
			return names, nil
		}

		rows := make([]struct {
			ID   pulid.ID `bun:"id"`
			Name string   `bun:"name"`
		}, 0, len(ids))
		cols := documenttypecol.DocumentTypeColumns
		if err := f.db.DBForContext(ctx).
			NewSelect().
			Model((*documenttype.DocumentType)(nil)).
			ColumnExpr(cols.ID.Qualified()).
			ColumnExpr(cols.Name.Qualified()).
			Apply(documenttypecol.DocumentTypeApplyTenant(tenantInfo)).
			Where(cols.ID.In(), bun.List(ids)).
			Scan(ctx, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			names[row.ID] = row.Name
		}

		return names, nil
	})
}
