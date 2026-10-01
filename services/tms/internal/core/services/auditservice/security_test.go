package auditservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type capturingAuditService struct {
	services.AuditService

	params *services.LogActionParams
	entry  *audit.Entry
	err    error
}

func (c *capturingAuditService) LogAction(params *services.LogActionParams, opts ...services.LogOption) error {
	c.params = params
	c.entry = &audit.Entry{Metadata: map[string]any{}}
	for _, opt := range opts {
		if err := opt(c.entry); err != nil {
			return err
		}
	}
	return c.err
}

func newSecurityAuditor(svc services.AuditService) services.SecurityAuditor {
	return NewSecurityAuditor(SecurityAuditorParams{
		AuditService: svc,
		Metrics:      &metrics.Registry{Audit: metrics.NewAudit(nil, zap.NewNop(), false)},
		Logger:       zap.NewNop(),
	})
}

func TestSecurityAuditorRecordsACriticalEntryWithTheRequest(t *testing.T) {
	t.Parallel()

	capture := &capturingAuditService{}
	auditor := newSecurityAuditor(capture)

	tenant := pulidTenant()
	actor := services.UserActor(tenant).AuditActor()
	ctx := requestmeta.With(t.Context(), requestmeta.New("req-7", "192.0.2.10", "admin-browser"))

	auditor.RecordChange(ctx, services.SecurityChange{
		Resource:       permission.ResourceAPIKey,
		ResourceID:     "key_1",
		Operation:      permission.OpUpdate,
		Actor:          actor,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Before:         map[string]any{"status": "active"},
		After:          map[string]any{"status": "revoked"},
		Comment:        "API key revoked",
		Metadata:       map[string]any{"reason": "rotation"},
	})

	require.NotNil(t, capture.params)
	assert.True(t, capture.params.Critical)
	assert.Equal(t, actor.UserID, capture.params.UserID)
	assert.Equal(t, services.PrincipalTypeUser, capture.params.PrincipalType)
	assert.Equal(t, tenant.OrgID, capture.params.OrganizationID)
	assert.Equal(t, "active", capture.params.PreviousState["status"])
	assert.Equal(t, "revoked", capture.params.CurrentState["status"])
	assert.Equal(t, audit.CategoryUser, capture.entry.Category)
	assert.Equal(t, "192.0.2.10", capture.entry.IPAddress)
	assert.Equal(t, "admin-browser", capture.entry.UserAgent)
	assert.Equal(t, "req-7", capture.entry.CorrelationID)
	assert.Equal(t, "API key revoked", capture.entry.Comment)
	assert.Equal(t, "rotation", capture.entry.Metadata["reason"])
	assert.NotEmpty(t, capture.entry.Changes)
}

func TestSecurityAuditorAcceptsListStates(t *testing.T) {
	t.Parallel()

	capture := &capturingAuditService{}
	auditor := newSecurityAuditor(capture)

	auditor.RecordChange(t.Context(), services.SecurityChange{
		Resource:  permission.ResourceUser,
		Operation: permission.OpUpdate,
		Before:    []string{"org_a"},
		After:     []string{"org_a", "org_b"},
	})

	require.NotNil(t, capture.params)
	assert.Equal(t, []any{"org_a"}, capture.params.PreviousState["value"])
	assert.Equal(t, []any{"org_a", "org_b"}, capture.params.CurrentState["value"])
}

func TestSecurityAuditorSurvivesAnUnserializableStateAndAFailedWrite(t *testing.T) {
	t.Parallel()

	capture := &capturingAuditService{err: errors.New("audit unavailable")}
	auditor := newSecurityAuditor(capture)

	assert.NotPanics(t, func() {
		auditor.RecordChange(context.WithoutCancel(t.Context()), services.SecurityChange{
			Resource:  permission.ResourceRole,
			Operation: permission.OpDelete,
			Before:    make(chan int),
		})
	})
	require.NotNil(t, capture.params)
	assert.Nil(t, capture.params.PreviousState)
}

func pulidTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}
