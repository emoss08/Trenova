package auditservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"go.uber.org/zap"
)

const (
	agentNameCacheSize = 1024
	agentNameCacheTTL  = 10 * time.Minute
	agentNameTimeout   = 2 * time.Second
)

type agentDefinitionReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
}

type agentNameKey struct {
	organizationID pulid.ID
	businessUnitID pulid.ID
	definitionID   pulid.ID
}

type agentCredits struct {
	definitions agentDefinitionReader
	names       *expirable.LRU[agentNameKey, string]
	logger      *zap.Logger
}

func newAgentCredits(definitions agentDefinitionReader, logger *zap.Logger) *agentCredits {
	return &agentCredits{
		definitions: definitions,
		names: expirable.NewLRU[agentNameKey, string](
			agentNameCacheSize,
			nil,
			agentNameCacheTTL,
		),
		logger: logger,
	}
}

func (c *agentCredits) credit(entry *audit.Entry) {
	if entry.PrincipalType != string(services.PrincipalTypeAgent) {
		return
	}

	entry.CreditAgent(c.nameOf(entry))
}

func (c *agentCredits) nameOf(entry *audit.Entry) string {
	if c == nil || c.definitions == nil ||
		entry.PrincipalID.Prefix() != agentdefinition.IDPrefix {
		return ""
	}

	key := agentNameKey{
		organizationID: entry.OrganizationID,
		businessUnitID: entry.BusinessUnitID,
		definitionID:   entry.PrincipalID,
	}
	if name, ok := c.names.Get(key); ok {
		return name
	}

	ctx, cancel := context.WithTimeout(context.Background(), agentNameTimeout)
	defer cancel()
	ctx = dbscope.WithValidTenant(ctx, dbscope.Tenant{
		OrganizationID: entry.OrganizationID,
		BusinessUnitID: entry.BusinessUnitID,
	})

	definition, err := c.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID: entry.PrincipalID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entry.OrganizationID,
			BuID:  entry.BusinessUnitID,
		},
	})
	if err != nil {
		c.logger.Warn("audit entry agent name unavailable",
			zap.String("agentDefinitionId", entry.PrincipalID.String()),
			zap.Error(err),
		)

		return ""
	}

	c.names.Add(key, definition.Name)

	return definition.Name
}
