package invoiceadjustmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapInvoiceAdjustmentPersistenceError(t *testing.T) {
	t.Run("maps correction group foreign key violations to business error", func(t *testing.T) {
		err := mapInvoiceAdjustmentPersistenceError(&pgconn.PgError{
			Code:           "23503",
			ConstraintName: "fk_invoice_adjustments_correction_group",
		})

		var businessErr *errortypes.BusinessError
		require.ErrorAs(t, err, &businessErr)
		assert.Contains(
			t,
			businessErr.Error(),
			"correction group is no longer valid",
		)
	})

	t.Run("passes through unrelated errors", func(t *testing.T) {
		input := &pgconn.PgError{
			Code:           "23503",
			ConstraintName: "fk_invoice_adjustments_original_invoice",
		}

		assert.Same(t, input, mapInvoiceAdjustmentPersistenceError(input))
	})
}
