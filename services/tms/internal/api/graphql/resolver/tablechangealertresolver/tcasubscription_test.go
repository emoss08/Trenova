package tablechangealertresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func subscriptionInput() gqlmodel.SaveTCASubscriptionInput {
	return gqlmodel.SaveTCASubscriptionInput{
		Name:           "Big loads",
		TableName:      "shipments",
		EventTypes:     []string{"INSERT"},
		Conditions:     []map[string]any{{"field": "weight", "operator": "gt", "value": 40000}},
		ConditionMatch: "all",
		WatchedColumns: []string{"weight"},
		Priority:       "high",
		Status:         "Active",
	}
}

func TestTCAResolvers_CheckTheTableChangeAlertPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"tcaAllowlistedTables", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).TcaAllowlistedTables(ctx)
			return err
		}},
		{"createTcaSubscription", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateTcaSubscription(ctx, subscriptionInput())
			return err
		}},
		{"updateTcaSubscription", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateTcaSubscription(
				ctx,
				pulid.MustNew("tcas_").String(),
				2,
				subscriptionInput(),
			)
			return err
		}},
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
				permission.ResourceTableChangeAlert,
				call.operation,
			)
		})
	}
}

func TestTCASubscriptionFromInput_MapsEverySetting(t *testing.T) {
	t.Parallel()

	recordID := "shp_01"
	input := subscriptionInput()
	input.RecordID = &recordID
	input.CustomTitle = "Heavy"

	entity, err := tcaSubscriptionFromInput(&input)
	require.NoError(t, err)

	assert.Equal(t, "Big loads", entity.Name)
	assert.Equal(t, "shipments", entity.TableName)
	assert.Equal(t, &recordID, entity.RecordID)
	assert.Equal(t, []string{"INSERT"}, entity.EventTypes)
	require.Len(t, entity.Conditions, 1)
	assert.Equal(t, "weight", entity.Conditions[0].Field)
	assert.Equal(t, tablechangealert.ConditionOperator("gt"), entity.Conditions[0].Operator)
	assert.Equal(t, "all", entity.ConditionMatch)
	assert.Equal(t, "Heavy", entity.CustomTitle)
	assert.Equal(t, "high", entity.Priority)
	assert.Equal(t, tablechangealert.SubscriptionStatusActive, entity.Status)
}

func TestTCASubscriptionFromInput_TreatsAnEmptyRecordAsWholeTable(t *testing.T) {
	t.Parallel()

	empty := ""
	input := subscriptionInput()
	input.RecordID = &empty

	entity, err := tcaSubscriptionFromInput(&input)
	require.NoError(t, err)
	assert.Nil(t, entity.RecordID)
}

func TestTCASubscriptionFromInput_RefusesAMalformedCondition(t *testing.T) {
	t.Parallel()

	input := subscriptionInput()
	input.Conditions = []map[string]any{{"field": 12}}

	_, err := tcaSubscriptionFromInput(&input)

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "conditions", fieldErr.Field)
}
