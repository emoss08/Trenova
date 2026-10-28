package factoringtools

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeNoas struct {
	factoringports.NoaService

	guard agenttoolstest.WriteGuard
	noas  map[pulid.ID]*factoring.Noa
	to    []string

	drafted      *factoringports.DraftNoaRequest
	sent         *factoringports.SendNoaRequest
	acknowledged *factoringports.AcknowledgeNoaRequest
	released     *factoringports.ReleaseNoaRequest
}

func (f *fakeNoas) Get(
	_ context.Context,
	req factoringports.GetNoaRequest,
) (*factoring.Noa, error) {
	noa, ok := f.noas[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Notice of assignment not found")
	}
	out := *noa
	return &out, nil
}

func (f *fakeNoas) PlanSend(
	ctx context.Context,
	req *factoringports.SendNoaRequest,
	_ pulid.ID,
) (*factoringports.NoaSendPlan, error) {
	noa, err := f.Get(ctx, factoringports.GetNoaRequest{ID: req.ID})
	if err != nil {
		return nil, err
	}
	if !noa.CanSend() {
		return nil, errortypes.NewBusinessError("Only a drafted or sent notice can be sent")
	}
	return &factoringports.NoaSendPlan{
		Noa: noa,
		Message: factoringports.MessagePlan{
			To:             f.to,
			Subject:        "Notice of assignment",
			HTML:           "<p>Please pay the factor.</p>",
			Text:           "Please pay the factor.",
			AttachmentName: "Notice-of-Assignment.pdf",
		},
	}, nil
}

func (f *fakeNoas) Draft(
	_ context.Context,
	req *factoringports.DraftNoaRequest,
	_ *permissionports.RequestActor,
) (*factoring.Noa, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.drafted = req
	return &factoring.Noa{}, nil
}

func (f *fakeNoas) Send(
	_ context.Context,
	req *factoringports.SendNoaRequest,
	_ *permissionports.RequestActor,
) (*factoring.Noa, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.sent = req
	return f.noas[req.ID], nil
}

func (f *fakeNoas) Acknowledge(
	_ context.Context,
	req *factoringports.AcknowledgeNoaRequest,
	_ *permissionports.RequestActor,
) (*factoring.Noa, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.acknowledged = req
	return f.noas[req.ID], nil
}

func (f *fakeNoas) Release(
	_ context.Context,
	req *factoringports.ReleaseNoaRequest,
	_ *permissionports.RequestActor,
) (*factoring.Noa, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.released = req
	return f.noas[req.ID], nil
}

type fakeReadiness struct {
	factoringports.ReadinessService

	guard    agenttoolstest.WriteGuard
	override *factoringports.OverridePlan
	request  *factoringports.DocumentRequestPlan

	overridden *factoringports.SetOverrideRequest
	requested  *factoringports.RequestMissingDocumentsRequest
	planned    *factoringports.RequestMissingDocumentsRequest
}

func (f *fakeReadiness) PlanOverride(
	_ context.Context,
	_ *factoringports.SetOverrideRequest,
) (*factoringports.OverridePlan, error) {
	return f.override, nil
}

func (f *fakeReadiness) SetOverride(
	_ context.Context,
	req *factoringports.SetOverrideRequest,
	_ *permissionports.RequestActor,
) (*factoring.InvoiceState, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.overridden = req
	return &factoring.InvoiceState{}, nil
}

func (f *fakeReadiness) PlanDocumentRequest(
	_ context.Context,
	req *factoringports.RequestMissingDocumentsRequest,
	_ pulid.ID,
) (*factoringports.DocumentRequestPlan, error) {
	f.planned = req
	return f.request, nil
}

func (f *fakeReadiness) RequestMissingDocuments(
	_ context.Context,
	req *factoringports.RequestMissingDocumentsRequest,
	_ *permissionports.RequestActor,
) (*factoringports.RequestMissingDocumentsResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.requested = req
	return &factoringports.RequestMissingDocumentsResult{}, nil
}

type fakeSenders struct {
	suppressed []string
}

func (f fakeSenders) ResolveSender(
	context.Context,
	*emailports.SendEmailRequest,
) (*emailports.EmailSender, error) {
	return &emailports.EmailSender{
		Email:      "billing@trenova.example",
		Name:       "Billing",
		Suppressed: f.suppressed,
	}, nil
}

func toolParams(params map[string]any) agenttoolports.ToolExecuteParams {
	out := agenttoolstest.ExecuteParams(params)
	out.IdempotencyKey = "call-1"
	out.Actor.PrincipalType = permissionports.PrincipalTypeUser
	return out
}

func noaWithStatus(status factoring.NoaStatus) *factoring.Noa {
	return &factoring.Noa{
		ID:                 pulid.MustNew("facnoa_"),
		CustomerID:         pulid.MustNew("cus_"),
		FactoringCompanyID: pulid.MustNew("facco_"),
		Status:             status,
		Version:            3,
	}
}

func newNoas(noa *factoring.Noa, to ...string) *fakeNoas {
	return &fakeNoas{noas: map[pulid.ID]*factoring.Noa{noa.ID: noa}, to: to}
}

func TestPolicies_KeepOutsideMailWithAPerson(t *testing.T) {
	t.Parallel()

	noas := &fakeNoas{}
	readiness := &fakeReadiness{}
	senders := fakeSenders{}
	cases := []struct {
		tool        agenttoolports.AgentTool
		resource    permission.Resource
		defaultTier agent.AutonomyTier
		maxTier     agent.AutonomyTier
		external    bool
	}{
		{
			NewDraftFactoringNoaTool(noas),
			permission.ResourceFactoringNoa,
			agent.TierAutoExecute,
			agent.TierAutoExecute,
			false,
		},
		{
			NewSendFactoringNoaTool(noas, senders),
			permission.ResourceFactoringNoa,
			agent.TierPropose,
			agent.TierPropose,
			true,
		},
		{
			NewRecordFactoringNoaAcknowledgementTool(noas),
			permission.ResourceFactoringNoa,
			agent.TierActWithApproval,
			agent.TierActWithApproval,
			false,
		},
		{
			NewReleaseFactoringNoaTool(noas),
			permission.ResourceFactoringNoa,
			agent.TierActWithApproval,
			agent.TierActWithApproval,
			false,
		},
		{
			NewSetInvoiceFactoringOverrideTool(readiness),
			permission.ResourceInvoice,
			agent.TierActWithApproval,
			agent.TierActWithApproval,
			false,
		},
		{
			NewRequestMissingFactoringDocumentTool(readiness, senders),
			permission.ResourceCustomerCommunication,
			agent.TierPropose,
			agent.TierPropose,
			true,
		},
	}

	for _, tc := range cases {
		policy := tc.tool.Policy()
		assert.Equal(t, tc.tool.Name(), policy.Name)
		assert.Equal(t, tc.resource, policy.Resource, tc.tool.Name())
		assert.Equal(t, tc.defaultTier, policy.DefaultTier, tc.tool.Name())
		assert.Equal(t, tc.maxTier, policy.MaxTier, tc.tool.Name())
		if tc.external {
			assert.Equal(
				t,
				[]agent.EgressClass{agent.EgressExternalRecipient},
				policy.Egress,
				tc.tool.Name(),
			)
		} else {
			assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress, tc.tool.Name())
		}
	}
}

func TestDraftFactoringNoa_DraftsForTheNamedCustomerAndFactor(t *testing.T) {
	t.Parallel()

	noas := &fakeNoas{}
	tool := NewDraftFactoringNoaTool(noas)
	customerID := pulid.MustNew("cus_")
	factorID := pulid.MustNew("facco_")
	params := toolParams(map[string]any{
		paramCustomerID: customerID.String(),
		paramFactorID:   factorID.String(),
		paramNote:       "Open invoices are included.",
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&noas.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceFactoringNoa, change.Resource)
	agenttoolstest.FieldByPath(t, change, "status")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, noas.drafted)
	assert.Equal(t, customerID, noas.drafted.CustomerID)
	assert.Equal(t, factorID, noas.drafted.FactoringCompanyID)
	assert.Equal(t, "Open invoices are included.", noas.drafted.Notes)
	assert.Equal(t, params.OrganizationID, noas.drafted.TenantInfo.OrgID)
}

func TestSendFactoringNoa_PreviewsTheLetterAndSendsThePinnedVersion(t *testing.T) {
	t.Parallel()

	noa := noaWithStatus(factoring.NoaStatusDraft)
	noas := newNoas(noa, "ap@customer.example")
	tool := NewSendFactoringNoaTool(noas, fakeSenders{})
	params := toolParams(map[string]any{
		paramNoaID:   noa.ID.String(),
		paramVersion: float64(noa.Version),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&noas.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "ap@customer.example")
	var message, update bool
	for i := range preview.Changes {
		change := &preview.Changes[i]
		if change.Message != nil {
			message = true
		}
		if change.Operation == agent.PreviewOperationUpdate {
			update = true
			agenttoolstest.FieldByPath(t, change, "status")
		}
	}
	assert.True(t, message, "the preview shows the email")
	assert.True(t, update, "the preview shows the notice marked sent")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, noas.sent)
	assert.Equal(t, noa.ID, noas.sent.ID)
	assert.Equal(t, noa.Version, noas.sent.Version)

	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(
		t,
		agenttoolports.ToolTarget{Resource: permission.ResourceFactoringNoa, ID: noa.ID},
		target,
	)
}

func TestSendFactoringNoa_WarnsAboutSuppressedRecipients(t *testing.T) {
	t.Parallel()

	noa := noaWithStatus(factoring.NoaStatusDraft)
	noas := newNoas(noa, "ap@customer.example")
	tool := NewSendFactoringNoaTool(noas, fakeSenders{suppressed: []string{"ap@customer.example"}})
	params := toolParams(map[string]any{
		paramNoaID:   noa.ID.String(),
		paramVersion: float64(noa.Version),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&noas.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.NotEmpty(t, preview.Warnings)
}

func TestSendFactoringNoa_RefusesAReleasedNotice(t *testing.T) {
	t.Parallel()

	noa := noaWithStatus(factoring.NoaStatusReleased)
	noas := newNoas(noa, "ap@customer.example")
	tool := NewSendFactoringNoaTool(noas, fakeSenders{})
	params := toolParams(map[string]any{
		paramNoaID:   noa.ID.String(),
		paramVersion: float64(noa.Version),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&noas.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.NotEmpty(t, preview.Warnings, "a notice that cannot be sent previews as a refusal")
	assert.Empty(t, preview.Changes)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
}

func TestRecordFactoringNoaAcknowledgement_RecordsTheSignedCopy(t *testing.T) {
	t.Parallel()

	noa := noaWithStatus(factoring.NoaStatusSent)
	noas := newNoas(noa)
	tool := NewRecordFactoringNoaAcknowledgementTool(noas)
	documentID := pulid.MustNew("doc_")
	signed := time.Now().Add(-48 * time.Hour).UTC()
	params := toolParams(map[string]any{
		paramNoaID:          noa.ID.String(),
		paramVersion:        float64(noa.Version),
		paramAcknowledgedAt: signed.Format(time.RFC3339),
		paramSignedDocument: documentID.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&noas.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	change := agenttoolstest.PreviewChange(t, preview, 0)
	agenttoolstest.FieldByPath(t, change, "status")
	agenttoolstest.FieldByPath(t, change, "signedDocumentId")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, noas.acknowledged)
	assert.Equal(t, documentID, noas.acknowledged.SignedDocumentID)
	assert.Equal(t, signed.Unix(), noas.acknowledged.AcknowledgedAt)
	assert.Nil(t, noas.acknowledged.ExpiresAt)
	assert.Equal(t, noa.Version, noas.acknowledged.Version)
}

func TestRecordFactoringNoaAcknowledgement_RefusesAFutureSignature(t *testing.T) {
	t.Parallel()

	noa := noaWithStatus(factoring.NoaStatusSent)
	noas := newNoas(noa)
	tool := NewRecordFactoringNoaAcknowledgementTool(noas)
	params := toolParams(map[string]any{
		paramNoaID:          noa.ID.String(),
		paramVersion:        float64(noa.Version),
		paramAcknowledgedAt: time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339),
		paramSignedDocument: pulid.MustNew("doc_").String(),
	})

	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&noas.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.NotEmpty(t, preview.Warnings)
}

func TestReleaseFactoringNoa_NeedsAReason(t *testing.T) {
	t.Parallel()

	noa := noaWithStatus(factoring.NoaStatusAcknowledged)
	noas := newNoas(noa)
	tool := NewReleaseFactoringNoaTool(noas)

	missing := toolParams(map[string]any{
		paramNoaID:   noa.ID.String(),
		paramVersion: float64(noa.Version),
	})
	require.Error(t, tool.Execute(t.Context(), missing))
	assert.Nil(t, noas.released)
	require.Error(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), missing))

	params := toolParams(map[string]any{
		paramNoaID:   noa.ID.String(),
		paramVersion: float64(noa.Version),
		paramReason:  "The factor's release letter arrived.",
	})
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&noas.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "releaseReason")

	require.NoError(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	assert.Nil(t, noas.released, "validating releases nothing")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, noas.released)
	assert.Equal(t, "The factor's release letter arrived.", noas.released.Reason)
}

func overridePlan(changed bool) *factoringports.OverridePlan {
	factorID := pulid.MustNew("facco_")
	before := &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		Number:             "INV-1001",
		FactoringOverride:  invoice.FactoringOverrideDefault,
		FactoringCompanyID: &factorID,
		Version:            2,
	}
	after := *before
	if changed {
		after.FactoringOverride = invoice.FactoringOverrideExclude
		after.FactoringCompanyID = nil
	}
	return &factoringports.OverridePlan{Before: before, After: &after, Changed: changed}
}

func TestSetInvoiceFactoringOverride_PreviewsTheExclusion(t *testing.T) {
	t.Parallel()

	plan := overridePlan(true)
	readiness := &fakeReadiness{override: plan}
	tool := NewSetInvoiceFactoringOverrideTool(readiness)
	params := toolParams(map[string]any{
		paramInvoiceID: plan.Before.ID.String(),
		paramOverride:  string(invoice.FactoringOverrideExclude),
		paramReason:    "Customer disputes the lane rate",
	})

	require.NoError(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&readiness.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "INV-1001")
	assert.Contains(t, preview.Summary, "Customer disputes the lane rate")
	change := agenttoolstest.PreviewChange(t, preview, 0)
	agenttoolstest.FieldByPath(t, change, fieldFactoringOverride)
	agenttoolstest.FieldByPath(t, change, fieldFactoringCompany)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, readiness.overridden)
	assert.Equal(t, invoice.FactoringOverrideExclude, readiness.overridden.Override)
	assert.Equal(t, plan.Before.ID, readiness.overridden.InvoiceID)
}

func TestSetInvoiceFactoringOverride_SaysWhenNothingChanges(t *testing.T) {
	t.Parallel()

	plan := overridePlan(false)
	readiness := &fakeReadiness{override: plan}
	tool := NewSetInvoiceFactoringOverrideTool(readiness)
	params := toolParams(map[string]any{
		paramInvoiceID: plan.Before.ID.String(),
		paramOverride:  string(invoice.FactoringOverrideDefault),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&readiness.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "nothing changes")
	assert.Empty(t, preview.Changes)
}

func documentPlan() *factoringports.DocumentRequestPlan {
	return &factoringports.DocumentRequestPlan{
		Invoice:       &invoice.Invoice{ID: pulid.MustNew("inv_"), Number: "INV-1002"},
		RecipientKind: "Driver",
		DocumentTypes: []string{"Proof of delivery"},
		ShipmentIDs:   []pulid.ID{pulid.MustNew("shp_")},
		Message: factoringports.MessagePlan{
			To:      []string{"driver@fleet.example"},
			Subject: "Documents needed for INV-1002",
			HTML:    "<p>Please send the proof of delivery.</p>",
			Text:    "Please send the proof of delivery.",
		},
	}
}

func TestRequestMissingFactoringDocument_PreviewsTheEmailToTheDriver(t *testing.T) {
	t.Parallel()

	plan := documentPlan()
	readiness := &fakeReadiness{request: plan}
	tool := NewRequestMissingFactoringDocumentTool(readiness, fakeSenders{})
	params := toolParams(map[string]any{paramInvoiceID: plan.Invoice.ID.String()})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&readiness.guard,
		func() (*agent.ToolPreview, error) {
			return tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
		},
	)
	assert.Contains(t, preview.Summary, "driver@fleet.example")
	assert.Contains(t, preview.Summary, "(driver)")
	assert.Contains(t, preview.Summary, "Proof of delivery")
	require.NotNil(t, agenttoolstest.PreviewChange(t, preview, 0).Message)
	require.NotNil(t, readiness.planned)
	assert.Nil(t, readiness.planned.DocumentTypeIDs, "no document types asks for every missing one")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, readiness.requested)
	assert.Equal(t, plan.Invoice.ID, readiness.requested.InvoiceID)
}

func TestRequestMissingFactoringDocument_RefusesMalformedDocumentTypes(t *testing.T) {
	t.Parallel()

	readiness := &fakeReadiness{request: documentPlan()}
	tool := NewRequestMissingFactoringDocumentTool(readiness, fakeSenders{})
	params := toolParams(map[string]any{
		paramInvoiceID:       pulid.MustNew("inv_").String(),
		paramDocumentTypeIDs: []any{"not-an-id"},
	})

	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, readiness.requested)

	typeID := pulid.MustNew("dt_")
	params.Params[paramDocumentTypeIDs] = []any{typeID.String()}
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, readiness.requested)
	assert.Equal(t, []pulid.ID{typeID}, readiness.requested.DocumentTypeIDs)
}
