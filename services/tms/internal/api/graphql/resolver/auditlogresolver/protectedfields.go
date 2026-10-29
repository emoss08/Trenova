package auditlogresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
)

// auditGroups is the protected field groups the reader of an audit entry may
// see. Every way GraphQL reaches an entry (the audit log, a record's history,
// an AI audit event's entries) resolves its states and changes here, so a
// protected field is withheld from the trail as from the record.
func (r *AuditEntryResolver) auditGroups(ctx context.Context) protectedfield.GroupSet {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok {
		return 0
	}
	return r.VisibleFieldGroups(ctx, authCtx)
}
