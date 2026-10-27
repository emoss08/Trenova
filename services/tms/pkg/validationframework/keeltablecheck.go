package validationframework

import (
	"context"
	"fmt"
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

var tableNamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

func KeelTableReferenceCheck(conn KeelDB, table string) CustomReferenceCheckFunc {
	if !tableNamePattern.MatchString(table) {
		panic(fmt.Sprintf("validationframework: invalid reference table name %q", table))
	}

	statement := `SELECT EXISTS (SELECT 1 FROM "` + table +
		`" WHERE id = $1 AND organization_id = $2 AND business_unit_id = $3)`

	return func(ctx context.Context, orgID, buID pulid.ID, refID pulid.ID) (bool, error) {
		if refID.IsNil() {
			return true, nil
		}
		if conn == nil {
			return false, errDBNotInitialized
		}

		var exists bool
		err := conn.RunScoped(ctx, true, func(ctx context.Context) error {
			found, err := keel.QueryValue[bool](
				ctx,
				conn.Keel(ctx),
				statement,
				refID.String(),
				orgID.String(),
				buID.String(),
			)
			if err != nil {
				return err
			}
			exists = found
			return nil
		})

		return exists, err
	}
}
