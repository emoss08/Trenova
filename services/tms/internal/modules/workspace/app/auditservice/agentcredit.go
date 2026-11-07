package auditservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/maypok86/otter/v2"
	"go.uber.org/zap"
)

const (
	agentNameTimeout   = 2 * time.Second
	agentNameCacheSize = 10_000
	agentNameTTL       = 10 * time.Minute
)

type agentDefinitionReader interface {
	GetByID(
		ctx context.Context,
		req agentdefinitionports.GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
}

type agentCredits struct {
	definitions agentDefinitionReader
	names       *otter.Cache[pulid.ID, string]
	logger      *zap.Logger
}

func newAgentCredits(definitions agentDefinitionReader, logger *zap.Logger) *agentCredits {
	return &agentCredits{
		definitions: definitions,
		names: otter.Must(&otter.Options[pulid.ID, string]{
			MaximumSize:      agentNameCacheSize,
			ExpiryCalculator: otter.ExpiryWriting[pulid.ID, string](agentNameTTL),
		}),
		logger: logger,
	}
}

func (c *agentCredits) credit(ctx context.Context, entry *audit.Entry) {
	if entry.PrincipalType != string(permissionports.PrincipalTypeAgent) {
		return
	}

	name := c.nameOf(ctx, entry)
	entry.CreditAgent(name)
	if entry.ActorName == "" && name != "" {
		entry.ActorName = name
	}
}

func (c *agentCredits) nameOf(parent context.Context, entry *audit.Entry) string {
	if c == nil || c.definitions == nil || entry.PrincipalID.Prefix() != agentdefinition.IDPrefix {
		return ""
	}
	if name, ok := c.names.GetIfPresent(entry.PrincipalID); ok {
		return name
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), agentNameTimeout)
	defer cancel()
	ctx = dbscope.WithValidTenant(ctx, dbscope.Tenant{
		OrganizationID: entry.OrganizationID,
		BusinessUnitID: entry.BusinessUnitID,
	})

	definition, err := c.definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID: entry.PrincipalID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entry.OrganizationID,
				BuID:  entry.BusinessUnitID,
			},
		},
	)
	if err != nil {
		c.logger.Warn("audit entry agent name unavailable",
			zap.String("agentDefinitionId", entry.PrincipalID.String()),
			zap.Error(err),
		)

		return ""
	}

	c.names.Set(entry.PrincipalID, definition.Name)
	return definition.Name
}
