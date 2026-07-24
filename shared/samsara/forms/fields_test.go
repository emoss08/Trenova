package forms

import (
	"testing"
	"time"

	samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"
	"github.com/stretchr/testify/assert"
)

func strPtr(v string) *string { return &v }

func TestFieldTypedValue(t *testing.T) {
	t.Parallel()

	dt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name      string
		field     FormField
		wantKind  string
		wantValue string
	}{
		{
			name: "text",
			field: FormField{
				Type:      samsaraspec.FormsFieldInputObjectResponseBodyTypeText,
				TextValue: &FieldTextValue{Value: "hello"},
			},
			wantKind:  FieldKindText,
			wantValue: "hello",
		},
		{
			name: "number integral",
			field: FormField{
				Type:        samsaraspec.FormsFieldInputObjectResponseBodyTypeNumber,
				NumberValue: &FieldNumberValue{Value: 42},
			},
			wantKind:  FieldKindNumber,
			wantValue: "42",
		},
		{
			name: "number fractional",
			field: FormField{
				Type:        samsaraspec.FormsFieldInputObjectResponseBodyTypeNumber,
				NumberValue: &FieldNumberValue{Value: 12.5},
			},
			wantKind:  FieldKindNumber,
			wantValue: "12.5",
		},
		{
			name: "multiple choice",
			field: FormField{
				Type: samsaraspec.FormsFieldInputObjectResponseBodyTypeMultipleChoice,
				MultipleChoiceValue: &FieldMultipleChoiceValue{
					Value:   "Option B",
					ValueId: "opt-2",
				},
			},
			wantKind:  FieldKindMultipleChoice,
			wantValue: "Option B",
		},
		{
			name: "check boxes",
			field: FormField{
				Type: samsaraspec.FormsFieldInputObjectResponseBodyTypeCheckBoxes,
				CheckBoxesValue: &FieldCheckBoxesValue{
					Value: []string{"Front", "Rear", "Brakes"},
				},
			},
			wantKind:  FieldKindCheckBoxes,
			wantValue: "Front, Rear, Brakes",
		},
		{
			name: "datetime",
			field: FormField{
				Type:          samsaraspec.FormsFieldInputObjectResponseBodyTypeDatetime,
				DateTimeValue: &FieldDateTimeValue{Value: dt},
			},
			wantKind:  FieldKindDateTime,
			wantValue: "2026-01-02T03:04:05Z",
		},
		{
			name: "signature",
			field: FormField{
				Type: samsaraspec.FormsFieldInputObjectResponseBodyTypeSignature,
				SignatureValue: &FieldSignatureValue{
					Media: MediaRecord{Url: strPtr("https://media/sig.png")},
				},
			},
			wantKind:  FieldKindSignature,
			wantValue: "https://media/sig.png",
		},
		{
			name: "signature no media url",
			field: FormField{
				Type:           samsaraspec.FormsFieldInputObjectResponseBodyTypeSignature,
				SignatureValue: &FieldSignatureValue{Media: MediaRecord{}},
			},
			wantKind:  FieldKindSignature,
			wantValue: "",
		},
		{
			name: "media first url",
			field: FormField{
				Type: samsaraspec.FormsFieldInputObjectResponseBodyTypeMedia,
				MediaValue: &FieldMediaValue{
					MediaList: []MediaRecord{
						{Url: nil},
						{Url: strPtr("https://media/photo.jpg")},
					},
				},
			},
			wantKind:  FieldKindMedia,
			wantValue: "https://media/photo.jpg",
		},
		{
			name: "asset name",
			field: FormField{
				Type: samsaraspec.FormsFieldInputObjectResponseBodyTypeAsset,
				AssetValue: &FieldAssetValue{
					Asset: samsaraspec.FormsAssetObjectResponseBody{
						Name: strPtr("Trailer 12"),
					},
				},
			},
			wantKind:  FieldKindAsset,
			wantValue: "Trailer 12",
		},
		{
			name: "person name",
			field: FormField{
				Type: samsaraspec.FormsFieldInputObjectResponseBodyTypePerson,
				PersonValue: &FieldPersonValue{
					Person: samsaraspec.FormsPersonObjectResponseBody{
						Name: strPtr("Alex Driver"),
					},
				},
			},
			wantKind:  FieldKindPerson,
			wantValue: "Alex Driver",
		},
		{
			name: "geofence name",
			field: FormField{
				Type: samsaraspec.FormsFieldInputObjectResponseBodyTypeGeofence,
				GeofenceValue: &FieldGeofenceValue{
					Geofence: samsaraspec.FormsGeofenceObjectResponseBody{
						Name: strPtr("Main Yard"),
					},
				},
			},
			wantKind:  FieldKindGeofence,
			wantValue: "Main Yard",
		},
		{
			name: "table",
			field: FormField{
				Type: samsaraspec.FormsFieldInputObjectResponseBodyTypeTable,
			},
			wantKind:  FieldKindTable,
			wantValue: "",
		},
		{
			name: "text missing value",
			field: FormField{
				Type: samsaraspec.FormsFieldInputObjectResponseBodyTypeText,
			},
			wantKind:  FieldKindText,
			wantValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kind, value := FieldTypedValue(tt.field)
			assert.Equal(t, tt.wantKind, kind)
			assert.Equal(t, tt.wantValue, value)
			assert.Equal(t, tt.wantValue, FieldDisplayValue(tt.field))
		})
	}
}

func TestSubmissionAccessors(t *testing.T) {
	t.Parallel()

	externalIDs := map[string]string{"shipmentId": "ship-9"}
	loc := SubmissionLocation{Latitude: 1.5, Longitude: -2.5}
	sub := FormSubmission{
		RouteStopId: strPtr("stop-1"),
		RouteId:     strPtr("route-1"),
		ExternalIds: &externalIDs,
		Location:    &loc,
	}

	assert.Equal(t, "stop-1", SubmissionRouteStopID(sub))
	assert.Equal(t, "route-1", SubmissionRouteID(sub))
	assert.Equal(t, externalIDs, SubmissionExternalIDs(sub))
	require := SubmissionLocationOf(sub)
	assert.NotNil(t, require)
	assert.InDelta(t, 1.5, require.Latitude, 1e-9)

	empty := FormSubmission{}
	assert.Empty(t, SubmissionRouteStopID(empty))
	assert.Empty(t, SubmissionRouteID(empty))
	assert.Nil(t, SubmissionExternalIDs(empty))
	assert.Nil(t, SubmissionLocationOf(empty))
}
