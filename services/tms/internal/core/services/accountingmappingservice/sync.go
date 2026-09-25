package accountingmappingservice

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) historyUse(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	row *accountingsync.AccountingMapping,
) (int, error) {
	if s.syncRecords == nil || row.State != accountingsync.MappingStateConfirmed {
		return 0, nil
	}
	usage, err := s.syncRecords.MappingUsage(ctx, &repositories.AccountingSyncMappingUsageRequest{
		TenantInfo:   tenantInfo,
		ConnectionID: row.ConnectionID,
		MappingIDs:   []pulid.ID{row.ID},
	})
	if err != nil {
		return 0, err
	}
	return usage[row.ID], nil
}

func (s *Service) guardHistory(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	row *accountingsync.AccountingMapping,
	acknowledged bool,
) (int, error) {
	used, err := s.historyUse(ctx, tenantInfo, row)
	if err != nil || used == 0 || acknowledged {
		return used, err
	}
	return used, errortypes.NewValidationError(
		"acknowledgeHistory",
		errortypes.ErrInvalidOperation,
		"{0} was used by {1} documents already sent to the accounting system. "+
			"They keep the record they were sent with; confirm that only new documents change",
		row.TargetLabel,
		used,
	)
}

func historyNote(used int) string {
	if used == 0 {
		return ""
	}
	return " (" + strconv.Itoa(used) + " documents already sent keep the previous record)"
}

func (s *Service) requeueMappingBlocked(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionIDs ...pulid.ID,
) {
	if s.syncRecords == nil {
		return
	}
	seen := make(map[pulid.ID]struct{}, len(connectionIDs))
	for _, connectionID := range connectionIDs {
		if _, ok := seen[connectionID]; ok || connectionID.IsNil() {
			continue
		}
		seen[connectionID] = struct{}{}

		requeued, err := s.syncRecords.Requeue(
			ctx,
			&repositories.RequeueAccountingSyncRecordsRequest{
				TenantInfo:   tenantInfo,
				ConnectionID: connectionID,
				ErrorCategories: []accountingsync.SyncErrorCategory{
					accountingsync.SyncErrorMapping,
				},
				At: timeutils.NowUnix(),
			},
		)
		if err != nil {
			s.l.Warn("failed to requeue records blocked on a mapping",
				zap.String("connectionId", connectionID.String()), zap.Error(err))
			continue
		}
		if requeued == 0 || s.dispatcher == nil {
			continue
		}
		if err = s.dispatcher.Kick(ctx, tenantInfo, connectionID); err != nil {
			s.l.Warn("failed to wake the accounting dispatcher",
				zap.String("connectionId", connectionID.String()), zap.Error(err))
		}
	}
}

func (s *Service) EnsureMapping(
	ctx context.Context,
	req *services.EnsureAccountingMappingRequest,
) (*accountingsync.AccountingMapping, error) {
	lookup := &repositories.GetAccountingMappingByTargetRequest{
		TenantInfo:      req.TenantInfo,
		ConnectionID:    req.ConnectionID,
		TargetType:      req.TargetType,
		TrenovaObjectID: req.ObjectID,
		TrenovaKey:      req.Key,
	}
	row, err := s.mappings.GetByTarget(ctx, lookup)
	if err == nil || !errortypes.IsNotFoundError(err) {
		return row, err
	}

	t, err := s.targetFor(ctx, req)
	if err != nil {
		return nil, err
	}
	if _, err = s.mappings.CreateMissing(ctx, []*accountingsync.AccountingMapping{{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		ConnectionID:    req.ConnectionID,
		TargetType:      t.TargetType,
		TrenovaObjectID: t.ObjectID,
		TrenovaKey:      t.Key,
		TargetLabel:     t.Label,
		ProviderKind:    t.TargetType.ProviderKind(),
		State:           accountingsync.MappingStateUnmatched,
		Signals:         accountingsync.MappingSignals{},
	}}); err != nil {
		return nil, err
	}
	return s.mappings.GetByTarget(ctx, lookup)
}

func (s *Service) targetFor(
	ctx context.Context,
	req *services.EnsureAccountingMappingRequest,
) (*target, error) {
	switch req.TargetType {
	case accountingsync.TargetCustomer:
		cus, err := s.customers.GetByID(ctx, repositories.GetCustomerByIDRequest{
			ID:         req.ObjectID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
		return customerTarget(cus), nil
	case accountingsync.TargetAccessorialCharge:
		tenantInfo := req.TenantInfo
		charge, err := s.accessorials.GetByID(ctx, repositories.GetAccessorialChargeByIDRequest{
			ID:         req.ObjectID,
			TenantInfo: &tenantInfo,
		})
		if err != nil {
			return nil, err
		}
		return accessorialTarget(charge), nil
	case accountingsync.TargetAccountRole,
		accountingsync.TargetLineType,
		accountingsync.TargetItemRole,
		accountingsync.TargetPaymentTerm,
		accountingsync.TargetPaymentMethod:
		if !req.TargetType.AcceptsKey(req.Key) {
			return nil, errortypes.NewValidationError(
				"key",
				errortypes.ErrInvalid,
				"{0} is not a {1} key",
				req.Key,
				string(req.TargetType),
			)
		}
		return &target{
			TargetType: req.TargetType,
			Key:        req.Key,
			Label:      keyLabel(req.TargetType, req.Key),
		}, nil
	case accountingsync.TargetCarrier:
		return nil, errortypes.NewBusinessError(
			"Carrier mappings are created by the reference refresh",
		)
	default:
		return nil, errortypes.NewValidationError(
			"targetType",
			errortypes.ErrInvalid,
			"{0} is not a mapping target",
			string(req.TargetType),
		)
	}
}
