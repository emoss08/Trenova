package accountingsyncservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

type mappingTarget struct {
	TargetType accountingsync.MappingTargetType
	ObjectID   pulid.ID
	Key        string
	Label      string
}

func (t mappingTarget) identity() string {
	if t.TargetType.KeyedByObject() {
		return string(t.TargetType) + "|" + t.ObjectID.String()
	}
	return string(t.TargetType) + "|" + t.Key
}

type resolver struct {
	s     *Service
	sess  *pushSession
	cache map[string]*accountingsync.AccountingMapping
	used  map[pulid.ID]struct{}
}

func newResolver(s *Service, sess *pushSession) *resolver {
	return &resolver{
		s:     s,
		sess:  sess,
		cache: make(map[string]*accountingsync.AccountingMapping, 8),
		used:  make(map[pulid.ID]struct{}, 8),
	}
}

func (r *resolver) mapping(
	ctx context.Context,
	target mappingTarget,
) (*accountingsync.AccountingMapping, error) {
	identity := target.identity()
	if cached, ok := r.cache[identity]; ok {
		return cached, nil
	}
	row, err := r.s.mappingService.EnsureMapping(ctx, &services.EnsureAccountingMappingRequest{
		TenantInfo:   r.sess.tenant,
		ConnectionID: r.sess.conn.ID,
		TargetType:   target.TargetType,
		ObjectID:     target.ObjectID,
		Key:          target.Key,
	})
	if err != nil {
		return nil, err
	}
	r.cache[identity] = row
	return row, nil
}

func (r *resolver) require(ctx context.Context, target mappingTarget) (string, error) {
	row, err := r.mapping(ctx, target)
	if err != nil {
		return "", err
	}
	if row.State != accountingsync.MappingStateConfirmed || row.ExternalID == "" {
		return "", r.missing(row, target)
	}
	r.used[row.ID] = struct{}{}
	return row.ExternalID, nil
}

func (r *resolver) optional(ctx context.Context, target mappingTarget) (string, error) {
	row, err := r.mapping(ctx, target)
	if err != nil {
		return "", err
	}
	if row.State != accountingsync.MappingStateConfirmed || row.ExternalID == "" {
		return "", nil
	}
	r.used[row.ID] = struct{}{}
	return row.ExternalID, nil
}

func (r *resolver) missing(
	row *accountingsync.AccountingMapping,
	target mappingTarget,
) *accountingsync.SyncError {
	label := target.Label
	if label == "" {
		label = row.TargetLabel
	}
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorMapping,
		Code:     row.ID.String(),
		Message:  label + " is not mapped",
		Resolution: "Map " + label + " to a " + r.sess.providerName + " " +
			providerKindLabel(row.ProviderKind),
	}
}

func providerKindLabel(kind accountingsync.ReferenceKind) string {
	switch kind {
	case accountingsync.ReferenceKindAccount:
		return "account"
	case accountingsync.ReferenceKindItem:
		return "item"
	case accountingsync.ReferenceKindCustomer:
		return "customer"
	case accountingsync.ReferenceKindVendor:
		return "vendor"
	case accountingsync.ReferenceKindTerm:
		return "term"
	case accountingsync.ReferenceKindPaymentMethod:
		return "payment method"
	default:
		return "record"
	}
}

func (r *resolver) mappingIDs() []string {
	ids := make([]string, 0, len(r.used))
	for id := range r.used {
		ids = append(ids, id.String())
	}
	slices.Sort(ids)
	return ids
}
