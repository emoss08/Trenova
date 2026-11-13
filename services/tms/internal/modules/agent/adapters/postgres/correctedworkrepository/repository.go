package correctedworkrepository

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
)

const (
	maxAgentWrites = 2000
	maxPersonEdits = 1000
	recordsPerRead = 500
	maxValueChars  = 120
	// correctedWithin is how long after an agent's write a hand edit still
	// reads as a correction of it, not as the record moving on.
	correctedWithin = int64(7 * 24 * 60 * 60)
	changeUpdated   = "updated"
	changeKeyFrom   = "from"
	changeKeyTo     = "to"
	changeKeyType   = "type"
	fieldUpdatedAt  = "updatedAt"
	fieldVersion    = "version"
	fieldCreatedAt  = "createdAt"
	fieldSearchText = "searchVector"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) agentreflectionports.CorrectedWorkRepository {
	return &repository{db: p.DB}
}

// ownResources are the records agents keep about their own work. A person
// changing one of them is managing agents, not correcting what an agent did
// to the organization's records.
func ownResources() []permission.Resource {
	return []permission.Resource{
		permission.ResourceAgentRun,
		permission.ResourceAgentProposal,
		permission.ResourceAgentException,
		permission.ResourceAgentControl,
		permission.ResourceAgentExtension,
		permission.ResourceAgentDefinition,
		permission.ResourceAgentMemory,
		permission.ResourceAgentEvalSuite,
		permission.ResourceAgentFeedback,
		permission.ResourceAIProvider,
		permission.ResourceAIAuditTrail,
		permission.ResourceAssistant,
		permission.ResourceWatchtower,
	}
}

// ListCorrectedWork reads the agents' newest writes first and then, for the
// records they touched, what people changed by hand afterwards. Both reads
// are bounded: an organization whose agents wrote more than the bound in the
// stretch is read for its newest work, which is where a habit shows.
func (r *repository) ListCorrectedWork(
	ctx context.Context,
	req *agentreflectionports.ListCorrectedWorkRequest,
) ([]agentreflectionports.CorrectedWork, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentreflectionports.CorrectedWork, error) {
			ae := auditdb.Entries
			writes, err := ae.Pick(
				ae.ResourceID, ae.Resource, ae.Timestamp, ae.Operation, ae.Changes, ae.Metadata,
			).
				Where(
					ae.AIRunID.IsNotNull(),
					ae.SensitiveData.IsFalse(),
					ae.Metadata.HasKey(audit.MetadataAgentID),
					ae.Timestamp.Between(req.WroteSince, req.Until),
					ae.Operation.In(permission.OpCreate, permission.OpUpdate),
					ae.Resource.NotIn(ownResources()...),
				).
				Order(ae.Timestamp.Desc()).
				Limit(maxAgentWrites).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list what agents wrote: %w", err)
			}
			if len(writes) == 0 {
				return []agentreflectionports.CorrectedWork{}, nil
			}

			records := make([]string, 0, len(writes))
			for _, write := range writes {
				if !slices.Contains(records, write.ResourceID) {
					records = append(records, write.ResourceID)
				}
			}

			edits := make([]*audit.Entry, 0, 64)
			for chunk := range slices.Chunk(records, recordsPerRead) {
				if len(edits) >= maxPersonEdits {
					break
				}
				rows, listErr := ae.Pick(
					ae.ResourceID, ae.Resource, ae.Timestamp, ae.Changes, ae.EntityRef,
				).
					Where(
						ae.ResourceID.In(chunk...),
						ae.Timestamp.Between(req.ChangedSince, req.Until),
						ae.PrincipalType.Eq(string(permissionports.PrincipalTypeUser)),
						ae.AIRunID.IsNull(),
						ae.SensitiveData.IsFalse(),
						ae.Operation.Eq(permission.OpUpdate),
					).
					Order(ae.Timestamp.Desc()).
					Limit(maxPersonEdits-len(edits)).
					List(ctx, r.db.Keel(ctx))
				if listErr != nil {
					return nil, fmt.Errorf("list what people changed of agents' work: %w", listErr)
				}
				edits = append(edits, rows...)
			}

			return Corrected(writes, edits), nil
		},
	)
}

// Corrected pairs each hand edit with the agent's latest write to the same
// record before it, and keeps the fields of the agent's that the person
// changed. An agent that created the record set all of it; one that updated
// it set only what its change names.
func Corrected(writes, edits []*audit.Entry) []agentreflectionports.CorrectedWork {
	out := make([]agentreflectionports.CorrectedWork, 0, len(edits))
	for _, edit := range edits {
		write := latestBefore(writes, edit)
		if write == nil || edit.Timestamp-write.Timestamp > correctedWithin {
			continue
		}
		agentID, _ := write.Metadata[audit.MetadataAgentID].(string)
		if agentID == "" {
			continue
		}

		fields := make([]agentreflectionports.CorrectedField, 0, len(edit.Changes))
		for field, raw := range edit.Changes {
			if bookkeeping(field) {
				continue
			}
			if _, set := write.Changes[field]; !set && write.Operation != permission.OpCreate {
				continue
			}
			change, ok := raw.(map[string]any)
			if !ok || change[changeKeyType] != changeUpdated {
				continue
			}
			fields = append(fields, agentreflectionports.CorrectedField{
				Field: field,
				Was:   valueText(change[changeKeyFrom]),
				Now:   valueText(change[changeKeyTo]),
			})
		}
		if len(fields) == 0 {
			continue
		}
		slices.SortFunc(fields, func(a, b agentreflectionports.CorrectedField) int {
			return strings.Compare(a.Field, b.Field)
		})

		out = append(out, agentreflectionports.CorrectedWork{
			AgentDefinitionID: pulid.ID(agentID),
			Resource:          string(edit.Resource),
			ResourceID:        edit.ResourceID,
			Label:             edit.EntityRef,
			Fields:            fields,
			WrittenAt:         write.Timestamp,
			ChangedAt:         edit.Timestamp,
		})
	}

	return out
}

func latestBefore(writes []*audit.Entry, edit *audit.Entry) *audit.Entry {
	var latest *audit.Entry
	for _, write := range writes {
		if write.ResourceID != edit.ResourceID || write.Timestamp >= edit.Timestamp {
			continue
		}
		if latest == nil || write.Timestamp > latest.Timestamp {
			latest = write
		}
	}

	return latest
}

func bookkeeping(field string) bool {
	switch field {
	case fieldUpdatedAt, fieldVersion, fieldCreatedAt, fieldSearchText:
		return true
	default:
		return false
	}
}

func valueText(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return stringutils.Ellipsize(v, maxValueChars)
	default:
		encoded, err := sonic.MarshalString(v)
		if err != nil {
			return ""
		}

		return stringutils.Ellipsize(encoded, maxValueChars)
	}
}
