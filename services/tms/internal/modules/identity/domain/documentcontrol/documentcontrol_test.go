package documentcontrol_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultControl() *documentcontrol.DocumentControl {
	return documentcontrol.NewDefaultDocumentControl(pulid.MustNew("org_"), pulid.MustNew("bu_"))
}

func validate(control *documentcontrol.DocumentControl) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	control.Validate(multiErr)
	return multiErr
}

/*
The client reads the stored list against the four canonical names. A value
saved in another case or as an alias of a shipment is a box the settings page
cannot tick, so every value is stored by its canonical name, once, in the order
it was first chosen.
*/
func TestDocumentControl_NormalizeStoresCanonicalResources(t *testing.T) {
	t.Parallel()

	control := defaultControl()
	control.ShipmentDraftAllowedResources = []string{
		" Trailer ",
		"shipment_import",
		"INBOUND_MESSAGE",
		"assistant_thread",
		"SHIPMENT",
		"worker",
		"trailer",
	}

	control.Normalize()

	assert.Equal(
		t,
		[]string{"trailer", "shipment", "worker"},
		control.ShipmentDraftAllowedResources,
	)
	assert.False(t, validate(control).HasErrors())
}

func TestDocumentControl_NormalizeKeepsAnEmptyListEmpty(t *testing.T) {
	t.Parallel()

	control := defaultControl()
	control.EnableShipmentDraftExtraction = false
	control.ShipmentDraftAllowedResources = nil

	control.Normalize()

	assert.NotNil(t, control.ShipmentDraftAllowedResources)
	assert.Empty(t, control.ShipmentDraftAllowedResources)
}

func TestDocumentControl_ValidateRefusesUnknownResource(t *testing.T) {
	t.Parallel()

	control := defaultControl()
	control.ShipmentDraftAllowedResources = []string{"shipment", "Boat"}
	control.Normalize()

	multiErr := validate(control)

	require.Len(t, multiErr.Errors, 1)
	assert.True(t, multiErr.HasErrorAt("shipmentDraftAllowedResources"))
	assert.Equal(t, "unsupported resource type", multiErr.Errors[0].Message)
}

func TestDocumentControl_ValidateRefusesNonCanonicalResourceThatWasNotNormalized(t *testing.T) {
	t.Parallel()

	control := defaultControl()
	control.ShipmentDraftAllowedResources = []string{"Shipment"}

	multiErr := validate(control)

	require.Len(t, multiErr.Errors, 1)
	assert.True(t, multiErr.HasErrorAt("shipmentDraftAllowedResources"))
	assert.Equal(t, "unsupported resource type", multiErr.Errors[0].Message)
}

/*
Drafting switched on with no record kind to draft from is a switch that is on
and does nothing. The client refuses it with this message on this field; the
server refuses it the same way for any other caller.
*/
func TestDocumentControl_ValidateRefusesDraftingWithNoResources(t *testing.T) {
	t.Parallel()

	control := defaultControl()
	control.EnableShipmentDraftExtraction = true
	control.ShipmentDraftAllowedResources = []string{}

	multiErr := validate(control)

	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "shipmentDraftAllowedResources", multiErr.Errors[0].Field())
	assert.Equal(t, errortypes.ErrRequired, multiErr.Errors[0].Code)
	assert.Equal(t, "Choose at least one kind of record to draft from", multiErr.Errors[0].Message)
}

func TestDocumentControl_ValidateAcceptsNoResourcesWhileDraftingIsOff(t *testing.T) {
	t.Parallel()

	control := defaultControl()
	control.EnableShipmentDraftExtraction = false
	control.ShipmentDraftAllowedResources = []string{}

	assert.False(t, validate(control).HasErrors())
}

func TestDocumentControl_BeforeCreateNormalizesResources(t *testing.T) {
	t.Parallel()

	control := defaultControl()
	control.ShipmentDraftAllowedResources = []string{"SHIPMENT_IMPORT", "Tractor"}

	require.NoError(t, control.BeforeCreate(t.Context()))

	assert.Equal(t, []string{"shipment", "tractor"}, control.ShipmentDraftAllowedResources)
}

func TestDocumentControl_AsksAIToClassifyOnlyWhenBothSwitchesAreOn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		recognizes bool
		asksAI     bool
		want       bool
	}{
		{name: "both on", recognizes: true, asksAI: true, want: true},
		{name: "ask AI off", recognizes: true, asksAI: false, want: false},
		{name: "recognition off", recognizes: false, asksAI: true, want: false},
		{name: "both off", recognizes: false, asksAI: false, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			control := defaultControl()
			control.EnableAutoClassification = tt.recognizes
			control.EnableAIAssistedClassification = tt.asksAI

			assert.Equal(t, tt.want, control.AsksAIToClassify())
		})
	}
}
