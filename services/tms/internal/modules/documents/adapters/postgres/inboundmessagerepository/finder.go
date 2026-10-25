package inboundmessagerepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
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
	_ pagination.TenantInfo,
	reference string,
) (pulid.ID, bool, error) {
	return dbtx.Read2(ctx, f.db, func(ctx context.Context) (pulid.ID, bool, error) {
		reference = strings.TrimSpace(reference)
		if reference == "" {
			return pulid.Nil, false, nil
		}

		sp := shipmentdb.Shipments
		forms := referenceForms(reference)
		ids, err := keel.Into[pulid.ID](ctx, f.db.Keel(ctx), sp.
			Pick(sp.ID).
			Where(keel.Or(sp.ProNumber.In(forms...), sp.BOL.In(forms...))).
			Limit(2))
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

func lowerLike(col keel.Expr, pattern string) keel.Pred {
	return keel.DynFunc("lower", col).Like(pattern)
}

func (f *RecordFinder) FindCustomerByEmail(
	ctx context.Context,
	_ pagination.TenantInfo,
	address string,
) (pulid.ID, bool, error) {
	return dbtx.Read2(ctx, f.db, func(ctx context.Context) (pulid.ID, bool, error) {
		address = stringutils.NormalizeEmailAddress(address)
		if address == "" {
			return pulid.Nil, false, nil
		}

		db := f.db.Keel(ctx)
		pattern := "%" + stringutils.EscapeLikePattern(address) + "%"
		found := make(map[pulid.ID]struct{}, 2)

		type customerRecipients struct {
			ID         pulid.ID
			Recipients string
		}

		c := customerdb.Customers
		customers, err := keel.Into[customerRecipients](ctx, db, c.
			Pick(c.ID, keel.DynOf(c.StatusUpdateRecipients).Coalesce("")).
			Where(lowerLike(c.StatusUpdateRecipients, pattern)).
			Limit(maxPartyCandidates))
		if err != nil {
			return pulid.Nil, false, err
		}
		for _, row := range customers {
			if listsAddress(row.Recipients, address) {
				found[row.ID] = struct{}{}
			}
		}

		type profileRecipients struct {
			CustomerID pulid.ID
			To         string
			CC         string
		}

		p := customerdb.CustomerEmailProfiles
		profiles, err := keel.Into[profileRecipients](ctx, db, p.
			Pick(
				p.CustomerID,
				keel.DynOf(p.ToRecipients).Coalesce(""),
				keel.DynOf(p.CCRecipients).Coalesce(""),
			).
			Where(keel.Or(lowerLike(p.ToRecipients, pattern), lowerLike(p.CCRecipients, pattern))).
			Limit(maxPartyCandidates))
		if err != nil {
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
	_ pagination.TenantInfo,
	address string,
) (pulid.ID, bool, error) {
	return dbtx.Read2(ctx, f.db, func(ctx context.Context) (pulid.ID, bool, error) {
		address = stringutils.NormalizeEmailAddress(address)
		if address == "" {
			return pulid.Nil, false, nil
		}

		db := f.db.Keel(ctx)
		pattern := stringutils.EscapeLikePattern(address)
		found := make(map[pulid.ID]struct{}, 2)

		c := carrierdb.Carriers
		carriers, err := keel.Into[pulid.ID](ctx, db, c.
			Pick(c.ID).
			Where(lowerLike(c.Email, pattern)).
			Limit(maxPartyCandidates))
		if err != nil {
			return pulid.Nil, false, err
		}
		for _, id := range carriers {
			found[id] = struct{}{}
		}

		cc := carrierdb.CarrierContacts
		contacts, err := keel.Into[pulid.ID](ctx, db, cc.
			Pick(cc.CarrierID).
			Where(lowerLike(cc.Email, pattern)).
			Limit(maxPartyCandidates))
		if err != nil {
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
	_ pagination.TenantInfo,
	id pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, f.db, func(ctx context.Context) (bool, error) {
		sp := shipmentdb.Shipments
		return sp.Where(sp.ID.Eq(id)).Exists(ctx, f.db.Keel(ctx))
	})
}

func (f *RecordFinder) CustomerExists(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, f.db, func(ctx context.Context) (bool, error) {
		c := customerdb.Customers
		return c.Where(c.ID.Eq(id)).Exists(ctx, f.db.Keel(ctx))
	})
}

func (f *RecordFinder) CarrierExists(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (bool, error) {
	return dbtx.Read(ctx, f.db, func(ctx context.Context) (bool, error) {
		c := carrierdb.Carriers
		return c.Where(c.ID.Eq(id)).Exists(ctx, f.db.Keel(ctx))
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
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, f.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		names := make(map[pulid.ID]string, len(ids))
		if len(ids) == 0 {
			return names, nil
		}

		type typeName struct {
			ID   pulid.ID
			Name string
		}

		dt := documenttypedb.DocumentTypes
		rows, err := keel.Into[typeName](ctx, f.db.Keel(ctx), dt.
			Pick(dt.ID, dt.Name).
			Where(dt.ID.In(ids...)).
			Limit(len(ids)))
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			names[row.ID] = row.Name
		}

		return names, nil
	})
}
