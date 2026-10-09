package deskcase_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func keys(items []*deskcase.Item) []deskcase.ItemKey {
	out := make([]deskcase.ItemKey, 0, len(items))
	for _, entry := range items {
		out = append(out, entry.Key)
	}

	return out
}

func withMode(
	items deskcase.TemplateItems,
	key deskcase.ItemKey,
	mode deskcase.ItemMode,
) deskcase.TemplateItems {
	out := append(deskcase.TemplateItems(nil), items...)
	for idx := range out {
		if out[idx].Key == key {
			out[idx].Mode = mode
		}
	}

	return out
}

func TestReadyToBill_FollowsTheTemplatesOrderAndLeavesOutWhatIsOff(t *testing.T) {
	t.Parallel()

	template := deskcase.DefaultItems(deskcase.ChecklistReadyToBill)
	template[0], template[len(template)-1] = template[len(template)-1], template[0]
	template = withMode(template, deskcase.ItemCustomerNotified, deskcase.ModeOff)
	facts := readyShipment()
	facts.Template = template

	got := keys(deskcase.ReadyToBill(facts).Items)

	assert.Equal(t, deskcase.ItemBillingHolds, got[0])
	assert.Equal(t, deskcase.ItemDelivered, got[len(got)-1])
	assert.NotContains(t, got, deskcase.ItemCustomerNotified)
}

func TestReadyToBill_AnOptionalStepNeverBlocksNorComesNext(t *testing.T) {
	t.Parallel()

	facts := readyShipment()
	facts.CustomerNotifiedAt = nil
	facts.Template = withMode(
		deskcase.DefaultItems(deskcase.ChecklistReadyToBill),
		deskcase.ItemCustomerNotified,
		deskcase.ModeOptional,
	)

	got := deskcase.ReadyToBill(facts)

	notified := item(t, got, deskcase.ItemCustomerNotified)
	assert.Equal(t, deskcase.ItemBlocked, notified.State)
	assert.True(t, notified.Optional)
	assert.True(t, got.Ready)
	assert.Equal(t, deskcase.StepMarkReady, got.Next)
}

func TestReadyToBill_AddedSteps(t *testing.T) {
	t.Parallel()

	lumper := pulid.MustNew("dt_")
	manual := deskcase.ItemKey("custom:callshipper")
	document := deskcase.ItemKey("custom:lumperslip")
	template := append(deskcase.DefaultItems(deskcase.ChecklistReadyToBill),
		deskcase.TemplateItem{Key: manual, Mode: deskcase.ModeRequired, Custom: &deskcase.CustomItem{
			Label: "Shipper called", Check: deskcase.CheckManual, StepLabel: "Call the shipper",
			Prompt: "Draft what I should say to the shipper.",
		}},
		deskcase.TemplateItem{Key: document, Mode: deskcase.ModeRequired, Custom: &deskcase.CustomItem{
			Label: "Lumper receipt", Check: deskcase.CheckDocument, DocumentTypeID: lumper,
		}},
	)
	facts := readyShipment()
	facts.Template = template

	got := deskcase.ReadyToBill(facts)
	call := item(t, got, manual)
	assert.Equal(t, deskcase.ItemBlocked, call.State)
	assert.True(t, call.Manual)
	assert.Equal(t, "Call the shipper", call.StepLabel)
	assert.Equal(t, deskcase.StepKey(manual), call.Step)
	assert.Equal(t, deskcase.StepKey(manual), got.Next)
	assert.False(t, got.Ready)

	facts.Ticks = map[deskcase.ItemKey]deskcase.TickFact{manual: {At: now, By: "Dana"}}
	facts.DocumentTypesOnFile = map[pulid.ID]struct{}{lumper: {}}
	got = deskcase.ReadyToBill(facts)
	assert.Equal(t, "Dana", item(t, got, manual).TickedBy)
	assert.Equal(t, deskcase.ItemDone, item(t, got, document).State)
	assert.True(t, got.Ready)
}

func TestNormalize_BringsAnOldTemplateUpToDate(t *testing.T) {
	t.Parallel()

	stored := deskcase.TemplateItems{
		{Key: deskcase.ItemBillingHolds, Mode: deskcase.ModeRequired},
		{Key: deskcase.ItemDelivered, Mode: deskcase.ModeOptional},
		{Key: deskcase.ItemKey("retired"), Mode: deskcase.ModeRequired},
		{Key: deskcase.ItemKey("custom:orphaned"), Mode: deskcase.ModeRequired},
	}

	got := deskcase.Normalize(deskcase.ChecklistReadyToBill, stored)

	require.Len(t, got, len(deskcase.DefaultItems(deskcase.ChecklistReadyToBill)))
	assert.Equal(t, deskcase.ItemBillingHolds, got[0].Key, "the saved order is kept")
	assert.Equal(t, deskcase.ModeRequired, got[1].Mode, "a locked step is always required")
	for _, entry := range got {
		assert.NotEqual(t, deskcase.ItemKey("retired"), entry.Key)
		assert.NotEqual(t, deskcase.ItemKey("custom:orphaned"), entry.Key,
			"an added step with nothing describing it is dropped")
	}
}

func TestChecklistTemplate_Validate(t *testing.T) {
	t.Parallel()

	valid := func() *deskcase.ChecklistTemplate {
		return &deskcase.ChecklistTemplate{
			Kind:  deskcase.ChecklistReadyToBill,
			Items: deskcase.DefaultItems(deskcase.ChecklistReadyToBill),
		}
	}
	errorsOf := func(template *deskcase.ChecklistTemplate) *errortypes.MultiError {
		multiErr := errortypes.NewMultiError()
		template.Validate(multiErr)
		return multiErr
	}

	assert.False(t, errorsOf(valid()).HasErrors())

	cases := map[string]func(template *deskcase.ChecklistTemplate){
		"a locked step turned off": func(template *deskcase.ChecklistTemplate) {
			template.Items = withMode(template.Items, deskcase.ItemDelivered, deskcase.ModeOff)
		},
		"a built-in step removed": func(template *deskcase.ChecklistTemplate) {
			template.Items = template.Items[1:]
		},
		"a step twice": func(template *deskcase.ChecklistTemplate) {
			template.Items = append(template.Items, template.Items[0])
		},
		"another checklist's step": func(template *deskcase.ChecklistTemplate) {
			template.Items = append(template.Items,
				deskcase.TemplateItem{Key: deskcase.ItemPaid, Mode: deskcase.ModeRequired})
		},
		"an added step without a name": func(template *deskcase.ChecklistTemplate) {
			template.Items = append(template.Items, deskcase.TemplateItem{
				Key: "custom:abcdef", Mode: deskcase.ModeRequired,
				Custom: &deskcase.CustomItem{Check: deskcase.CheckManual},
			})
		},
		"a document step without a document type": func(template *deskcase.ChecklistTemplate) {
			template.Items = append(template.Items, deskcase.TemplateItem{
				Key: "custom:abcdef", Mode: deskcase.ModeRequired,
				Custom: &deskcase.CustomItem{Label: "Lumper", Check: deskcase.CheckDocument},
			})
		},
		"a malformed added key": func(template *deskcase.ChecklistTemplate) {
			template.Items = append(template.Items, deskcase.TemplateItem{
				Key: "custom:A!", Mode: deskcase.ModeRequired,
				Custom: &deskcase.CustomItem{Label: "Lumper", Check: deskcase.CheckManual},
			})
		},
		"a document step on an invoice checklist": func(template *deskcase.ChecklistTemplate) {
			template.Kind = deskcase.ChecklistReadyToClose
			template.Items = append(deskcase.DefaultItems(deskcase.ChecklistReadyToClose),
				deskcase.TemplateItem{
					Key: "custom:abcdef", Mode: deskcase.ModeRequired,
					Custom: &deskcase.CustomItem{
						Label: "Remittance", Check: deskcase.CheckDocument,
						DocumentTypeID: pulid.MustNew("dt_"),
					},
				})
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			template := valid()
			mutate(template)
			assert.True(t, errorsOf(template).HasErrors())
		})
	}
}

func TestTickable(t *testing.T) {
	t.Parallel()

	key := deskcase.ItemKey("custom:callshipper")
	template := append(deskcase.DefaultItems(deskcase.ChecklistReadyToBill), deskcase.TemplateItem{
		Key: key, Mode: deskcase.ModeRequired,
		Custom: &deskcase.CustomItem{Label: "Shipper called", Check: deskcase.CheckManual},
	})

	assert.True(t, deskcase.Tickable(deskcase.ChecklistReadyToBill, template, key))
	assert.False(t, deskcase.Tickable(deskcase.ChecklistReadyToBill, template, deskcase.ItemPOD))
	assert.False(t, deskcase.Tickable(deskcase.ChecklistReadyToBill,
		withMode(template, key, deskcase.ModeOff), key))
	assert.False(t, deskcase.Tickable(deskcase.ChecklistReadyToBill, nil, key))
}
