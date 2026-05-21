package usageservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DocumentUploadUsageParams struct {
	TenantInfo     pagination.TenantInfo
	Actor          services.RequestActor
	DocumentID     pulid.ID
	IdempotencyKey string
	Quantity       int64
	CheckedAt      int64
	RecordedAt     int64
}

func CheckDocumentUploadLimit(
	ctx context.Context,
	provider services.UsageProvider,
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
	result, err := provider.CheckLimit(ctx, &services.UsageLimitCheckRequest{
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
		return errortypes.NewAuthorizationError(result.Reason)
	}

	return nil
}

func RecordDocumentUpload(
	ctx context.Context,
	provider services.UsageProvider,
	params DocumentUploadUsageParams,
) (*services.UsageRecordResult, error) {
	if provider == nil {
		return nil, nil
	}

	recordedAt := params.RecordedAt
	if recordedAt == 0 {
		recordedAt = time.Now().Unix()
	}

	actor := normalizeActor(params.TenantInfo, params.Actor)
	return provider.RecordUsage(ctx, &services.UsageRecordRequest{
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
	actor services.RequestActor,
) services.RequestActor {
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
		actor.PrincipalType = services.PrincipalTypeUser
	}
	if actor.PrincipalID.IsNil() {
		switch actor.PrincipalType {
		case services.PrincipalTypeAPIKey:
			actor.PrincipalID = actor.APIKeyID
		default:
			actor.PrincipalID = actor.UserID
		}
	}
	if actor.PrincipalType == services.PrincipalTypeAPIKey && actor.APIKeyID.IsNil() {
		actor.APIKeyID = actor.PrincipalID
	}

	return actor
}
