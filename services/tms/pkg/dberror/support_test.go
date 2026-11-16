package dberror

import (
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsSupportPaymentDestinationRefusal_KnowsTheGuardsCodeThroughWrapping(t *testing.T) {
	t.Parallel()

	refused := fmt.Errorf("save carrier: %w", &pgconn.PgError{Code: SupportPaymentDestinationCode})
	assert.True(t, IsSupportPaymentDestinationRefusal(refused))
	assert.False(t, IsRowLevelSecurityViolation(refused), "it is not read as a tenant boundary")

	assert.False(t, IsSupportPaymentDestinationRefusal(&pgconn.PgError{
		Code:    pgerrcode.InsufficientPrivilege,
		Message: "new row violates row-level security policy",
	}))
	assert.False(t, IsSupportPaymentDestinationRefusal(fmt.Errorf("plain")))
}

func TestIsSupportOutboundRefusal_KnowsOnlyTheOutboundGuardsCode(t *testing.T) {
	t.Parallel()

	refused := fmt.Errorf("queue email: %w", &pgconn.PgError{Code: SupportOutboundCode})
	assert.True(t, IsSupportOutboundRefusal(refused))
	assert.False(t, IsSupportPaymentDestinationRefusal(refused))
	assert.False(t, IsSupportOutboundRefusal(&pgconn.PgError{Code: SupportPaymentDestinationCode}))
}
