package inboundmessagerepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
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
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return pulid.Nil, false, nil
	}

	cols := buncolgen.ShipmentColumns
	forms := bun.In(referenceForms(reference))

	ids := make([]pulid.ID, 0, 2)
	err := f.db.DBForContext(ctx).
		NewSelect().
		Model((*shipment.Shipment)(nil)).
		ColumnExpr(cols.ID.Qualified()).
		Apply(buncolgen.ShipmentApplyTenant(tenantInfo)).
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

	customerCols := buncolgen.CustomerColumns
	var customers []customerRecipients
	if err := f.db.DBForContext(ctx).
		NewSelect().
		Model((*customer.Customer)(nil)).
		ColumnExpr(customerCols.ID.As("id")).
		ColumnExpr(customerCols.StatusUpdateRecipients.As("recipients")).
		Apply(buncolgen.CustomerApplyTenant(tenantInfo)).
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

	profileCols := buncolgen.CustomerEmailProfileColumns
	var profiles []profileRecipients
	if err := f.db.DBForContext(ctx).
		NewSelect().
		Model((*customer.CustomerEmailProfile)(nil)).
		ColumnExpr(profileCols.CustomerID.As("customer_id")).
		ColumnExpr(profileCols.ToRecipients.As("to_recipients")).
		ColumnExpr(profileCols.CCRecipients.As("cc_recipients")).
		Apply(buncolgen.CustomerEmailProfileApplyTenant(tenantInfo)).
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
	address = stringutils.NormalizeEmailAddress(address)
	if address == "" {
		return pulid.Nil, false, nil
	}
	found := make(map[pulid.ID]struct{}, 2)

	carrierCols := buncolgen.CarrierColumns
	carriers := make([]pulid.ID, 0, 2)
	if err := f.db.DBForContext(ctx).
		NewSelect().
		Model((*carrier.Carrier)(nil)).
		ColumnExpr(carrierCols.ID.Qualified()).
		Apply(buncolgen.CarrierApplyTenant(tenantInfo)).
		Where(carrierCols.Email.LowerLike(), stringutils.EscapeLikePattern(address)).
		Limit(maxPartyCandidates).
		Scan(ctx, &carriers); err != nil {
		return pulid.Nil, false, err
	}
	for _, id := range carriers {
		found[id] = struct{}{}
	}

	contactCols := buncolgen.CarrierContactColumns
	contacts := make([]pulid.ID, 0, 2)
	if err := f.db.DBForContext(ctx).
		NewSelect().
		Model((*carrier.CarrierContact)(nil)).
		ColumnExpr(contactCols.CarrierID.Qualified()).
		Apply(buncolgen.CarrierContactApplyTenant(tenantInfo)).
		Where(contactCols.Email.LowerLike(), stringutils.EscapeLikePattern(address)).
		Limit(maxPartyCandidates).
		Scan(ctx, &contacts); err != nil {
		return pulid.Nil, false, err
	}
	for _, id := range contacts {
		found[id] = struct{}{}
	}

	return onlyOneOf(found)
}

func (f *RecordFinder) ShipmentExists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (bool, error) {
	return f.db.DBForContext(ctx).
		NewSelect().
		Model((*shipment.Shipment)(nil)).
		Where(buncolgen.ShipmentColumns.ID.Eq(), id).
		Apply(buncolgen.ShipmentApplyTenant(tenantInfo)).
		Exists(ctx)
}

func (f *RecordFinder) CustomerExists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (bool, error) {
	return f.db.DBForContext(ctx).
		NewSelect().
		Model((*customer.Customer)(nil)).
		Where(buncolgen.CustomerColumns.ID.Eq(), id).
		Apply(buncolgen.CustomerApplyTenant(tenantInfo)).
		Exists(ctx)
}

func (f *RecordFinder) CarrierExists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (bool, error) {
	return f.db.DBForContext(ctx).
		NewSelect().
		Model((*carrier.Carrier)(nil)).
		Where(buncolgen.CarrierColumns.ID.Eq(), id).
		Apply(buncolgen.CarrierApplyTenant(tenantInfo)).
		Exists(ctx)
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
