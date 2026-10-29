package accountingsyncpush

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/shared/pulid"
)

type MappingTarget struct {
	TargetType accountingsync.MappingTargetType
	ObjectID   pulid.ID
	Key        string
	Label      string
}

func (t MappingTarget) Identity() string {
	if t.TargetType.KeyedByObject() {
		return string(t.TargetType) + "|" + t.ObjectID.String()
	}
	return string(t.TargetType) + "|" + t.Key
}

type Resolver struct {
	mappings accountingmappingports.AccountingMappingService
	sess     *Session
	cache    map[string]*accountingmapping.AccountingMapping
	used     map[pulid.ID]struct{}
}

func (k *Kit) NewResolver(sess *Session) *Resolver {
	return &Resolver{
		mappings: k.mappingService,
		sess:     sess,
		cache:    make(map[string]*accountingmapping.AccountingMapping, 8),
		used:     make(map[pulid.ID]struct{}, 8),
	}
}

func (r *Resolver) Use(row *accountingmapping.AccountingMapping) {
	r.used[row.ID] = struct{}{}
}

func (r *Resolver) Mapping(
	ctx context.Context,
	target MappingTarget,
) (*accountingmapping.AccountingMapping, error) {
	identity := target.Identity()
	if cached, ok := r.cache[identity]; ok {
		return cached, nil
	}
	row, err := r.mappings.EnsureMapping(
		ctx,
		&accountingmappingports.EnsureAccountingMappingRequest{
			TenantInfo:   r.sess.Tenant,
			ConnectionID: r.sess.Conn.ID,
			TargetType:   target.TargetType,
			ObjectID:     target.ObjectID,
			Key:          target.Key,
		},
	)
	if err != nil {
		return nil, err
	}
	r.cache[identity] = row
	return row, nil
}

func (r *Resolver) Require(ctx context.Context, target MappingTarget) (string, error) {
	row, err := r.Mapping(ctx, target)
	if err != nil {
		return "", err
	}
	if row.State != accountingmapping.MappingStateConfirmed || row.ExternalID == "" {
		return "", r.Missing(row, target)
	}
	r.used[row.ID] = struct{}{}
	return row.ExternalID, nil
}

func (r *Resolver) Optional(ctx context.Context, target MappingTarget) (string, error) {
	row, err := r.Mapping(ctx, target)
	if err != nil {
		return "", err
	}
	if row.State != accountingmapping.MappingStateConfirmed || row.ExternalID == "" {
		return "", nil
	}
	r.used[row.ID] = struct{}{}
	return row.ExternalID, nil
}

func (r *Resolver) Missing(
	row *accountingmapping.AccountingMapping,
	target MappingTarget,
) *accountingsync.SyncError {
	label := target.Label
	if label == "" {
		label = row.TargetLabel
	}
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorMapping,
		Code:     row.ID.String(),
		Message:  label + " is not mapped",
		Resolution: "Map " + label + " to a " + r.sess.ProviderName + " " +
			ProviderKindLabel(row.ProviderKind),
	}
}

func ProviderKindLabel(kind accountingprovider.ReferenceKind) string {
	switch kind {
	case accountingprovider.ReferenceKindAccount:
		return "account"
	case accountingprovider.ReferenceKindItem:
		return "item"
	case accountingprovider.ReferenceKindCustomer:
		return "customer"
	case accountingprovider.ReferenceKindVendor:
		return "vendor"
	case accountingprovider.ReferenceKindTerm:
		return "term"
	case accountingprovider.ReferenceKindPaymentMethod:
		return "payment method"
	default:
		return "record"
	}
}

func (r *Resolver) MappingIDs() []string {
	ids := make([]string, 0, len(r.used))
	for id := range r.used {
		ids = append(ids, id.String())
	}
	slices.Sort(ids)
	return ids
}

type AccountRefRequest struct {
	AccountID pulid.ID
	Label     string
	Role      string
	Missing   string
}

func (r *Resolver) Account(
	ctx context.Context,
	req *AccountRefRequest,
) (string, error) {
	if req.AccountID.IsNil() {
		return "", Blocked(
			accountingsync.SyncErrorConfiguration,
			req.Missing,
			"Set the default account in the accounting settings, then retry",
		)
	}

	target := MappingTarget{
		TargetType: accountingsync.TargetGLAccount,
		ObjectID:   req.AccountID,
		Label:      req.Label,
	}
	row, err := r.Mapping(ctx, target)
	if err != nil {
		return "", err
	}
	if row.State == accountingmapping.MappingStateConfirmed && row.ExternalID != "" {
		r.Use(row)
		return row.ExternalID, nil
	}

	if req.Role != "" {
		externalID, roleErr := r.Optional(ctx, MappingTarget{
			TargetType: accountingsync.TargetAccountRole,
			Key:        req.Role,
		})
		if roleErr != nil {
			return "", roleErr
		}
		if externalID != "" {
			return externalID, nil
		}
	}
	return "", r.Missing(row, target)
}
