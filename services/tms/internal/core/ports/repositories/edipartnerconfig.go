package repositories

import (
    "context"

    "github.com/emoss08/trenova/internal/core/domain/edi"
    "github.com/emoss08/trenova/shared/pulid"
)

// EDIPartnerConfigRepository provides access to persisted EDI partner configs.
type EDIPartnerConfigRepository interface {
    // GetByID fetches a partner config by its ID within a specific BU/Org.
    GetByID(ctx context.Context, buID, orgID pulid.ID, id pulid.ID) (*edi.PartnerConfig, error)
    // GetByKey fetches a partner config by its unique name within BU/Org.
    GetByKey(ctx context.Context, buID, orgID pulid.ID, name string) (*edi.PartnerConfig, error)
    // List returns up to limit partner configs, using keyset pagination with (afterName, afterID).
    List(ctx context.Context, buID, orgID pulid.ID, limit int, afterName string, afterID pulid.ID) ([]*edi.PartnerConfig, string, error)
}

