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
	"go.uber.org/zap"
)

const agentNameTimeout = 2 * time.Second

type agentDefinitionReader interface {
	GetByID(
		ctx context.Context,
		req agentdefinitionports.GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
}

type agentCredits struct {
	definitions agentDefinitionReader
	logger      *zap.Logger
}

func newAgentCredits(definitions agentDefinitionReader, logger *zap.Logger) *agentCredits {
	return &agentCredits{
		definitions: definitions,
		logger:      logger,
	}
}

func (c *agentCredits) credit(entry *audit.Entry) {
	if entry.PrincipalType != string(permissionports.PrincipalTypeAgent) {
		return
	}

	entry.CreditAgent(c.nameOf(entry))
}

func (c *agentCredits) nameOf(entry *audit.Entry) string {
	if c == nil || c.definitions == nil ||
		entry.PrincipalID.Prefix() != agentdefinition.IDPrefix {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), agentNameTimeout)
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

	return definition.Name
}
