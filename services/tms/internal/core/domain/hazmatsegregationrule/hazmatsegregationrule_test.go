package hazmatsegregationrule

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestHazmatSegregationRule_Validate_DistanceRequiresFields(t *testing.T) {
	t.Parallel()

	entity := &HazmatSegregationRule{
		Name:            "Distance Rule",
		ClassA:          hazardousmaterial.HazardousClass3,
		ClassB:          hazardousmaterial.HazardousClass8,
		SegregationType: SegregationTypeDistance,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	assert.True(t, multiErr.HasErrors())
}

func TestHazmatSegregationRule_Validate_HasExceptionsRequiresNotes(t *testing.T) {
	t.Parallel()

	entity := &HazmatSegregationRule{
		Name:            "Exception Rule",
		ClassA:          hazardousmaterial.HazardousClass3,
		ClassB:          hazardousmaterial.HazardousClass8,
		SegregationType: SegregationTypeProhibited,
		HasExceptions:   true,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	assert.True(t, multiErr.HasErrors())
}

func TestHazmatSegregationRule_Validate_SameClassRequiresDistinctSpecificMaterials(t *testing.T) {
	t.Parallel()

	entity := &HazmatSegregationRule{
		Name:            "Same Class Rule",
		ClassA:          hazardousmaterial.HazardousClass3,
		ClassB:          hazardousmaterial.HazardousClass3,
		SegregationType: SegregationTypeProhibited,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	assert.True(t, multiErr.HasErrors())
}
