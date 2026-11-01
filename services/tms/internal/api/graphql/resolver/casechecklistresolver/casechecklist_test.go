package casechecklistresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/deskcase"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaseChecklistResolvers_CheckTheBillingControlPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"caseChecklistTemplates", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).CaseChecklistTemplates(
				ctx,
				deskcase.ChecklistReadyToBill,
			)
			return err
		}},
		{
			"saveCaseChecklistTemplate",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).SaveCaseChecklistTemplate(
					ctx,
					gqlmodel.SaveCaseChecklistTemplateInput{Kind: deskcase.ChecklistReadyToBill},
				)
				return err
			},
		},
		{
			"deleteCaseChecklistTemplate",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).DeleteCaseChecklistTemplate(
					ctx,
					pulid.MustNew("cct_").String(),
				)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceBillingControl,
				call.operation,
			)
		})
	}
}

func TestSaveTemplateRequestFromInput_MapsTheTemplate(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("cct_")
	customerID := pulid.MustNew("cus_")
	documentTypeID := pulid.MustNew("dt_")
	rawTemplateID, rawCustomerID, rawDocumentTypeID := templateID.String(), customerID.String(),
		documentTypeID.String()
	prompt := "Check the lumper receipt"

	req, err := saveTemplateRequestFromInput(&gqlmodel.SaveCaseChecklistTemplateInput{
		ID:         &rawTemplateID,
		Version:    4,
		Kind:       deskcase.ChecklistReadyToClose,
		CustomerID: &rawCustomerID,
		Items: []*gqlmodel.CaseChecklistTemplateItemInput{
			{Key: "pod", Mode: deskcase.ModeRequired},
			{
				Key:  "lumper",
				Mode: deskcase.ModeOptional,
				Custom: &gqlmodel.CaseChecklistCustomItemInput{
					Label:          "Lumper receipt",
					Check:          deskcase.CustomCheck("Document"),
					DocumentTypeID: &rawDocumentTypeID,
					Prompt:         &prompt,
				},
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, templateID, req.ID)
	assert.Equal(t, customerID, req.CustomerID)
	assert.Equal(t, int64(4), req.Version)
	assert.Equal(t, deskcase.ChecklistReadyToClose, req.Kind)
	require.Len(t, req.Items, 2)
	assert.Equal(t, deskcase.ItemKey("pod"), req.Items[0].Key)
	assert.Nil(t, req.Items[0].Custom)
	require.NotNil(t, req.Items[1].Custom)
	assert.Equal(t, documentTypeID, req.Items[1].Custom.DocumentTypeID)
	assert.Equal(t, "Check the lumper receipt", req.Items[1].Custom.Prompt)
	assert.Empty(t, req.Items[1].Custom.StepLabel)
}

func TestSaveTemplateRequestFromInput_NamesAMalformedCustomer(t *testing.T) {
	t.Parallel()

	raw := "acme"
	_, err := saveTemplateRequestFromInput(&gqlmodel.SaveCaseChecklistTemplateInput{
		Kind:       deskcase.ChecklistReadyToBill,
		CustomerID: &raw,
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.NotEmpty(t, multiErr.Errors)
	assert.Equal(t, "customerId", multiErr.Errors[0].Field)
}

func TestCaseChecklistTemplateResolvers_RenderItemsAndLockedSteps(t *testing.T) {
	t.Parallel()

	template := &deskcase.ChecklistTemplate{Items: deskcase.TemplateItems{
		{Key: "pod", Mode: deskcase.ModeRequired},
	}}
	items, err := (&CaseChecklistTemplateResolver{&Deps{}}).Items(t.Context(), template)
	require.NoError(t, err)
	require.Len(t, items, 1)

	key, err := (&CaseChecklistTemplateItemResolver{&Deps{}}).Key(t.Context(), items[0])
	require.NoError(t, err)
	assert.Equal(t, "pod", key)

	locked, err := (&CaseChecklistTemplatesResolver{&Deps{}}).Locked(
		t.Context(),
		&deskcaseports.CaseChecklistTemplates{Locked: []deskcase.ItemKey{"rateConfirmation"}},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"rateConfirmation"}, locked)
}
