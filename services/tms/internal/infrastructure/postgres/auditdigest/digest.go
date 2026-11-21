package auditdigest

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
)

const digestSQL = `
SELECT r.retention_class,
	count(*)::bigint,
	encode(sha256(convert_to(string_agg(r.row_hash, '' ORDER BY r."timestamp", r.id), 'UTF8')), 'hex')
FROM (
	SELECT e.retention_class, e."timestamp", e.id,
		encode(sha256(convert_to(jsonb_build_array(
			e.id, e.organization_id, e.business_unit_id, e."timestamp", e.resource, e.resource_id,
			e.operation, e.action, e.category, e.retention_class, e.channel, e.principal_type,
			e.principal_id, e.user_id, e.api_key_id, e.actor_name, e.ai_run_id, e.entity_ref,
			e.correlation_id, e.trace_id, e.ip_address, e.user_agent, e.comment, e.changes,
			e.previous_state, e.current_state, e.metadata, e.sensitive_data, e.critical
		)::text, 'UTF8')), 'hex') AS row_hash
	FROM audit_entries e
	WHERE e.organization_id = $1
		AND e.business_unit_id = $2
		AND e."timestamp" >= $3
		AND e."timestamp" < $4
) r
GROUP BY r.retention_class`

type Window struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	From           int64
	To             int64
}

type Digest struct {
	Classes  map[string]audit.ClassDigest
	RowCount int64
}

func Compute(ctx context.Context, db *keel.DB, w Window) (*Digest, error) {
	out := &Digest{Classes: make(map[string]audit.ClassDigest, len(audit.RetentionClasses))}
	err := keel.QueryRows(ctx, db, func(row pgx.CollectableRow) error {
		var class, digest string
		var count int64
		if err := row.Scan(&class, &count, &digest); err != nil {
			return err
		}
		out.Classes[class] = audit.ClassDigest{Count: count, Digest: digest}
		out.RowCount += count
		return nil
	}, digestSQL, w.OrganizationID.String(), w.BusinessUnitID.String(), w.From, w.To)
	if err != nil {
		return nil, fmt.Errorf("digest audit entries: %w", err)
	}
	return out, nil
}
