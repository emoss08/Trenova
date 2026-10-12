package usageservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DocumentUploadUsageParams struct {
	TenantInfo     pagination.TenantInfo
	Actor          permissionports.RequestActor
	DocumentID     pulid.ID
	IdempotencyKey string
	Quantity       int64
	CheckedAt      int64
	RecordedAt     int64
}

func CheckDocumentUploadLimit(
	ctx context.Context,
	provider platformcatalogports.UsageProvider,
	params DocumentUploadUsageParams,
) error {
	if provider == nil {
		return nil
	}

	quantity := normalizeQuantity(params.Quantity)
	checkedAt := params.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	actor := normalizeActor(params.TenantInfo, params.Actor)
	result, err := provider.CheckLimit(ctx, &platformcatalogports.UsageLimitCheckRequest{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		MeterKey:       platformcatalog.MeterDocumentUploads,
		Quantity:       quantity,
		CheckedAt:      checkedAt,
		IdempotencyKey: documentUploadIdempotencyKey(params),
	})
	if err != nil {
		return err
	}
	if !result.Allowed {
		return LimitError(result)
	}

	return nil
}

type DocumentBytesUsageParams struct {
	TenantInfo pagination.TenantInfo
	Actor      permissionports.RequestActor
	Bytes      int64
	CheckedAt  int64
}

func CheckDocumentBytesLimit(
	ctx context.Context,
	provider platformcatalogports.UsageProvider,
	params DocumentBytesUsageParams,
) error {
	if provider == nil || params.Bytes <= 0 {
		return nil
	}

	checkedAt := params.CheckedAt
	if checkedAt == 0 {
		checkedAt = time.Now().Unix()
	}

	actor := normalizeActor(params.TenantInfo, params.Actor)
	for _, meter := range []platformcatalog.MeterKey{
		platformcatalog.MeterDocumentFileBytes,
		platformcatalog.MeterDocumentStorageBytes,
	} {
		result, err := provider.CheckLimit(ctx, &platformcatalogports.UsageLimitCheckRequest{
			OrganizationID: actor.OrganizationID,
			BusinessUnitID: actor.BusinessUnitID,
			PrincipalType:  actor.PrincipalType,
			PrincipalID:    actor.PrincipalID,
			UserID:         actor.UserID,
			APIKeyID:       actor.APIKeyID,
			MeterKey:       meter,
			Quantity:       params.Bytes,
			CheckedAt:      checkedAt,
		})
		if err != nil {
			return err
		}
		if !result.Allowed {
			return LimitError(result)
		}
	}

	return nil
}

func LimitError(result *platformcatalogports.UsageLimitCheckResult) error {
	if result.Reason == ReasonQuotaExceeded {
		return errortypes.NewQuotaExceededError(
			string(result.MeterKey),
			result.Limit,
			result.Used,
			result.Plan,
		)
	}

	return errortypes.NewAuthorizationError(result.Reason)
}

func RecordDocumentUpload(
	ctx context.Context,
	provider platformcatalogports.UsageProvider,
	params DocumentUploadUsageParams,
) (*platformcatalogports.UsageRecordResult, error) {
	if provider == nil {
		return nil, nil
	}

	recordedAt := params.RecordedAt
	if recordedAt == 0 {
		recordedAt = time.Now().Unix()
	}

	actor := normalizeActor(params.TenantInfo, params.Actor)
	return provider.RecordUsage(ctx, &platformcatalogports.UsageRecordRequest{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		MeterKey:       platformcatalog.MeterDocumentUploads,
		Quantity:       normalizeQuantity(params.Quantity),
		RecordedAt:     recordedAt,
		IdempotencyKey: documentUploadIdempotencyKey(params),
	})
}

func normalizeQuantity(quantity int64) int64 {
	if quantity > 0 {
		return quantity
	}

	return 1
}

func documentUploadIdempotencyKey(params DocumentUploadUsageParams) string {
	if params.IdempotencyKey != "" {
		return params.IdempotencyKey
	}
	if params.DocumentID.IsNotNil() {
		return fmt.Sprintf("document-upload:%s", params.DocumentID.String())
	}

	return ""
}

func normalizeActor(
	tenantInfo pagination.TenantInfo,
	actor permissionports.RequestActor,
) permissionports.RequestActor {
	if actor.OrganizationID.IsNil() {
		actor.OrganizationID = tenantInfo.OrgID
	}
	if actor.BusinessUnitID.IsNil() {
		actor.BusinessUnitID = tenantInfo.BuID
	}
	if actor.UserID.IsNil() {
		actor.UserID = tenantInfo.UserID
	}
	if actor.PrincipalType == "" {
		actor.PrincipalType = permissionports.PrincipalTypeUser
	}
	if actor.PrincipalID.IsNil() {
		switch actor.PrincipalType {
		case permissionports.PrincipalTypeAPIKey:
			actor.PrincipalID = actor.APIKeyID
		default:
			actor.PrincipalID = actor.UserID
		}
	}
	if actor.PrincipalType == permissionports.PrincipalTypeAPIKey && actor.APIKeyID.IsNil() {
		actor.APIKeyID = actor.PrincipalID
	}

	return actor
}
