package inboundreplyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTemplateFillsKnownPlaceholders(t *testing.T) {
	t.Parallel()

	out := RenderTemplate(
		"Hi {first} — {PRO} for {customer} ({po}) is {status} on {lane}, ETA {eta}. {unknown} {sender}",
		inboundmessageports.ReplyContext{
			SenderFirstName: "Dana",
			SenderAddress:   "edi@partner.example",
			ProNumber:       "S-104251",
			PONumber:        "PO 88240",
			CustomerName:    "Crestline Retail",
			Status:          "running about two hours behind",
			ETA:             "16:30 today",
			Lane:            "Columbus → Louisville",
		},
	)

	assert.Equal(t,
		"Hi Dana — S-104251 for Crestline Retail (PO 88240) is running about two hours behind "+
			"on Columbus → Louisville, ETA 16:30 today. {unknown} edi@partner.example",
		out,
	)
}

func TestRenderTemplateFallsBackWhenUnknown(t *testing.T) {
	t.Parallel()

	out := RenderTemplate("Hi {first}, {pro} is {status}; ETA {eta}.", inboundmessageports.ReplyContext{})

	assert.Equal(t, "Hi there, your load is moving; ETA to follow.", out)
}

func TestServiceRenderTemplateUsesMatchedShipment(t *testing.T) {
	t.Parallel()

	f := newFixture()
	created, err := f.service.CreateTemplate(t.Context(), &CreateTemplateRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		Input:      TemplateInput{Name: "Running late", Body: "Hi {first} — {pro} is {status}. {lane} {unknown}"},
	})
	require.NoError(t, err)

	out, err := f.service.RenderTemplate(t.Context(), &RenderTemplateRequest{
		TenantInfo: testTenant,
		TemplateID: created.ID,
		MessageID:  f.message.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, "Hi Dana — S-104251 is running about two hours behind. Columbus → Louisville {unknown}", out)
}

func TestCreateTemplateRejectsDuplicateName(t *testing.T) {
	t.Parallel()

	f := newFixture()
	_, err := f.service.CreateTemplate(t.Context(), &CreateTemplateRequest{
		TenantInfo: testTenant,
		Input:      TemplateInput{Name: "Running late", Body: "Body"},
	})
	require.NoError(t, err)

	_, err = f.service.CreateTemplate(t.Context(), &CreateTemplateRequest{
		TenantInfo: testTenant,
		Input:      TemplateInput{Name: " running LATE ", Body: "Other"},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsMultiError(err) || errortypes.IsError(err))
	assert.Contains(t, err.Error(), "already exists")
}

func TestCreateTemplateRequiresNameAndBody(t *testing.T) {
	t.Parallel()

	f := newFixture()
	_, err := f.service.CreateTemplate(t.Context(), &CreateTemplateRequest{
		TenantInfo: testTenant,
		Input:      TemplateInput{Name: " ", Body: " "},
	})
	require.Error(t, err)
	assert.Empty(t, f.templates.templates)
}

func TestCreateTemplateEnforcesLimit(t *testing.T) {
	t.Parallel()

	f := newFixture()
	for range triage.MaxTemplatesPerTenant {
		id := pulid.MustNew("ttpl_")
		f.templates.templates[id] = &triage.ReplyTemplate{ID: id, Name: id.String(), Body: "x"}
	}

	_, err := f.service.CreateTemplate(t.Context(), &CreateTemplateRequest{
		TenantInfo: testTenant,
		Input:      TemplateInput{Name: "One more", Body: "Body"},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestUpdateTemplateKeepsOwnName(t *testing.T) {
	t.Parallel()

	f := newFixture()
	created, err := f.service.CreateTemplate(t.Context(), &CreateTemplateRequest{
		TenantInfo: testTenant,
		Input:      TemplateInput{Name: "Running late", Body: "Body"},
	})
	require.NoError(t, err)

	updated, err := f.service.UpdateTemplate(t.Context(), &UpdateTemplateRequest{
		TenantInfo: testTenant,
		ID:         created.ID,
		Input:      TemplateInput{Name: "Running late", Body: "New body"},
	})
	require.NoError(t, err)
	assert.Equal(t, "New body", updated.Body)
}

func TestTemplateWriteErrorMapsUniqueViolation(t *testing.T) {
	t.Parallel()

	err := templateWriteError(uniqueViolation(), "Running late")
	assert.Contains(t, err.Error(), "already exists")
}
