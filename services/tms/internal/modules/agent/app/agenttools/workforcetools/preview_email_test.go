package workforcetools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailCustomerPreview_IsTheEmailExecuteSends(t *testing.T) {
	t.Parallel()

	tool, mailer, _, comments, _ := agenttoolstest.CustomerEmailFixture()
	tool.Deps.Senders = agenttoolstest.FakeSenders{}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "S12345 running about an hour late",
		"body":       "The truck is held in traffic; we now expect 3:30 PM.",
	})
	params.IdempotencyKey = "idem-1"

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, mailer.Sent, "a preview must not send")
	require.Nil(t, comments.Created, "a preview must not record")
	assert.Empty(t, preview.Warnings)

	send := agenttoolstest.FindChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, permission.ResourceCustomer, send.Resource)
	assert.Equal(t, "Acme Freight", send.Label)
	recorded := agenttoolstest.FindChange(t, preview, agent.PreviewOperationCreate)
	assert.Equal(t, permission.ResourceShipmentComment, recorded.Resource)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.AssertMessageIsTheSend(t, send.Message, mailer.Sent)
	assert.Equal(t, []string{"ops@acme.example", "ap@acme.example"}, send.Message.To)
	require.NotNil(t, comments.Created)
	assert.Equal(t, comments.Created.Comment, agenttoolstest.FindField(t, recorded, "comment").After)
}

func TestEmailCustomerPreview_SaysNothingGoesToACustomerAlreadyTold(t *testing.T) {
	t.Parallel()

	tool, mailer, _, comments, renderer := agenttoolstest.CustomerEmailFixture()
	comments.Existing = []*shipment.ShipmentComment{{
		CreatedAt: timeutils.NowUnix() - 600,
		Type:      shipment.CommentTypeCustomerUpdate,
		Metadata:  map[string]any{"source": "agent", "tool": "email_customer"},
	}}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "S12345 has departed",
		"body":       "The truck left the yard.",
	})
	params.IdempotencyKey = "idem-2"

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Empty(t, preview.Changes)
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agent.PreviewWarningAlreadyToldCustomer, preview.Warnings[0].Code)
	assert.Nil(t, renderer.Req, "nothing is rendered for a send that will not happen")

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrCustomerAlreadyTold)
	assert.Nil(t, mailer.Sent)
}

func TestEmailCustomerPreview_WarnsOfSuppressedRecipientsAndSensitiveText(t *testing.T) {
	t.Parallel()

	tool, _, _, _, _ := agenttoolstest.CustomerEmailFixture()
	tool.Deps.Senders = agenttoolstest.FakeSenders{
		Suppressed: []string{"ap@acme.example"},
		Refused:    true,
	}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "Remittance details",
		"body":       "Please remit to account 123456789 0001234567.",
	})
	params.IdempotencyKey = "idem-3"

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)

	codes := make([]agent.PreviewWarningCode, 0, len(preview.Warnings))
	for _, warning := range preview.Warnings {
		codes = append(codes, warning.Code)
	}
	assert.ElementsMatch(t, []agent.PreviewWarningCode{
		agent.PreviewWarningWouldFail,
		agent.PreviewWarningSensitiveContent,
	}, codes)
}

func TestEmailCustomerPreview_WarnsThatASuppressedCopyIsSkipped(t *testing.T) {
	t.Parallel()

	tool, _, _, _, _ := agenttoolstest.CustomerEmailFixture()
	tool.Deps.Senders = agenttoolstest.FakeSenders{Suppressed: []string{"cc@acme.example"}}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"profileId":  pulid.MustNew("emp_").String(),
		"subject":    "S12345 has departed",
		"body":       "The truck left the yard.",
	})
	params.IdempotencyKey = "idem-4"

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agent.PreviewWarningRecipientSkipped, preview.Warnings[0].Code)
	assert.Equal(t, []string{"cc@acme.example"}, preview.Warnings[0].Args)
}

func TestRequestMissingDocsPreview_IsTheEmailExecuteSends(t *testing.T) {
	t.Parallel()

	mailer := &agenttoolstest.FakeMailer{}
	tool := &RequestMissingDocsTool{
		email:     mailer,
		templates: &agenttoolstest.FakeRenderer{},
		senders:   agenttoolstest.FakeSenders{},
	}
	params := agenttoolstest.ExecuteParams(map[string]any{
		"profileId":          pulid.MustNew("emp_").String(),
		"to":                 []any{"pod@consignee.example"},
		"subject":            "Missing POD for SHP-1001",
		"body":               "We still need the signed delivery receipt.",
		"customerName":       "Halstead Grocery Group",
		"shipmentProNumber":  "SHP-1001",
		"requestedDocuments": []any{"Signed POD"},
	})
	params.IdempotencyKey = "idem-4"

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, mailer.Sent, "a preview must not send")
	assert.Contains(t, preview.Summary, "Signed POD")
	assert.Contains(t, preview.Summary, "SHP-1001")

	send := agenttoolstest.FindChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, "Halstead Grocery Group", send.Label)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.AssertMessageIsTheSend(t, send.Message, mailer.Sent)
}
