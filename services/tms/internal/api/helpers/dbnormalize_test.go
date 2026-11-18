package helpers_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeDatabaseError_APaymentDestinationRefusalReadsAsASentence(t *testing.T) {
	t.Parallel()

	raw := fmt.Errorf("update invoice: %w", &pgconn.PgError{
		Code:    dberror.SupportPaymentDestinationCode,
		Message: "Trenova support cannot change where money is sent (invoices.remittance_instructions)",
	})

	err := helpers.NormalizeDatabaseError(t.Context(), helpers.NewChainClassifier(), raw)

	var refusal *errortypes.AuthorizationError
	require.ErrorAs(t, err, &refusal)
	assert.Contains(t, refusal.Message, "where money is sent")
	assert.NotContains(t, refusal.Message, "invoices.", "the table and column stay in the logs")
}

func TestNormalizeDatabaseError_AnOutboundRefusalReadsAsASentence(t *testing.T) {
	t.Parallel()

	raw := &pgconn.PgError{Code: dberror.SupportOutboundCode, Message: "guarded (email_messages)"}
	err := helpers.NormalizeDatabaseError(t.Context(), helpers.NewChainClassifier(), raw)

	var refusal *errortypes.AuthorizationError
	require.ErrorAs(t, err, &refusal)
	assert.Contains(t, refusal.Message, "outside your organization")
}

func TestNormalizeDatabaseError_LeavesOtherDatabaseErrorsAsTheyWere(t *testing.T) {
	t.Parallel()

	raw := &pgconn.PgError{Code: "22001", Message: "value too long"}
	err := helpers.NormalizeDatabaseError(t.Context(), helpers.NewChainClassifier(), raw)
	assert.Same(t, raw, err)
}
