package forms

import (
	"strconv"
	"strings"
	"time"

	samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"
)

const (
	FieldKindText           = "text"
	FieldKindNumber         = "number"
	FieldKindMultipleChoice = "multipleChoice"
	FieldKindCheckBoxes     = "checkBoxes"
	FieldKindDateTime       = "dateTime"
	FieldKindSignature      = "signature"
	FieldKindMedia          = "media"
	FieldKindAsset          = "asset"
	FieldKindPerson         = "person"
	FieldKindGeofence       = "geofence"
	FieldKindTable          = "table"
	FieldKindUnknown        = "unknown"
)

func FieldDisplayValue(field FormField) string {
	_, value := FieldTypedValue(field)
	return value
}

func FieldTypedValue(field FormField) (kind string, value string) {
	switch field.Type {
	case samsaraspec.FormsFieldInputObjectResponseBodyTypeText:
		return FieldKindText, textFieldValue(field)
	case samsaraspec.FormsFieldInputObjectResponseBodyTypeNumber:
		return FieldKindNumber, numberFieldValue(field)
	case samsaraspec.FormsFieldInputObjectResponseBodyTypeMultipleChoice:
		return FieldKindMultipleChoice, multipleChoiceFieldValue(field)
	case samsaraspec.FormsFieldInputObjectResponseBodyTypeCheckBoxes:
		return FieldKindCheckBoxes, checkBoxesFieldValue(field)
	case samsaraspec.FormsFieldInputObjectResponseBodyTypeDatetime:
		return FieldKindDateTime, dateTimeFieldValue(field)
	case samsaraspec.FormsFieldInputObjectResponseBodyTypeSignature:
		return FieldKindSignature, signatureFieldValue(field)
	case samsaraspec.FormsFieldInputObjectResponseBodyTypeMedia:
		return FieldKindMedia, mediaFieldValue(field)
	case samsaraspec.FormsFieldInputObjectResponseBodyTypeAsset:
		return FieldKindAsset, assetFieldValue(field)
	case samsaraspec.FormsFieldInputObjectResponseBodyTypePerson:
		return FieldKindPerson, personFieldValue(field)
	case samsaraspec.FormsFieldInputObjectResponseBodyTypeGeofence:
		return FieldKindGeofence, geofenceFieldValue(field)
	case samsaraspec.FormsFieldInputObjectResponseBodyTypeTable:
		return FieldKindTable, ""
	default:
		return FieldKindUnknown, ""
	}
}

func textFieldValue(field FormField) string {
	if field.TextValue == nil {
		return ""
	}
	return field.TextValue.Value
}

func numberFieldValue(field FormField) string {
	if field.NumberValue == nil {
		return ""
	}
	return strconv.FormatFloat(field.NumberValue.Value, 'f', -1, 64)
}

func multipleChoiceFieldValue(field FormField) string {
	if field.MultipleChoiceValue == nil {
		return ""
	}
	return field.MultipleChoiceValue.Value
}

func checkBoxesFieldValue(field FormField) string {
	if field.CheckBoxesValue == nil {
		return ""
	}
	return strings.Join(field.CheckBoxesValue.Value, ", ")
}

func dateTimeFieldValue(field FormField) string {
	if field.DateTimeValue == nil {
		return ""
	}
	return field.DateTimeValue.Value.Format(time.RFC3339)
}

func signatureFieldValue(field FormField) string {
	if field.SignatureValue == nil {
		return ""
	}
	return mediaRecordURL(field.SignatureValue.Media)
}

func mediaFieldValue(field FormField) string {
	if field.MediaValue != nil {
		for i := range field.MediaValue.MediaList {
			if url := mediaRecordURL(field.MediaValue.MediaList[i]); url != "" {
				return url
			}
		}
	}
	if field.MediaList != nil {
		for i := range *field.MediaList {
			if url := mediaRecordURL((*field.MediaList)[i]); url != "" {
				return url
			}
		}
	}
	return ""
}

func assetFieldValue(field FormField) string {
	if field.AssetValue == nil {
		return ""
	}
	return derefString(field.AssetValue.Asset.Name, field.AssetValue.Asset.Id)
}

func personFieldValue(field FormField) string {
	if field.PersonValue == nil {
		return ""
	}
	person := field.PersonValue.Person
	if person.Name != nil && *person.Name != "" {
		return *person.Name
	}
	if person.PolymorphicUserId != nil {
		return person.PolymorphicUserId.Id
	}
	return ""
}

func geofenceFieldValue(field FormField) string {
	if field.GeofenceValue == nil {
		return ""
	}
	geofence := field.GeofenceValue.Geofence
	if geofence.Name != nil && *geofence.Name != "" {
		return *geofence.Name
	}
	if geofence.Address != nil && *geofence.Address != "" {
		return *geofence.Address
	}
	if geofence.Id != nil {
		return *geofence.Id
	}
	return ""
}

func mediaRecordURL(media MediaRecord) string {
	if media.Url == nil {
		return ""
	}
	return *media.Url
}

func derefString(values ...*string) string {
	for _, value := range values {
		if value != nil && *value != "" {
			return *value
		}
	}
	return ""
}

func SubmissionRouteStopID(s FormSubmission) string {
	if s.RouteStopId == nil {
		return ""
	}
	return *s.RouteStopId
}

func SubmissionRouteID(s FormSubmission) string {
	if s.RouteId == nil {
		return ""
	}
	return *s.RouteId
}

func SubmissionExternalIDs(s FormSubmission) map[string]string {
	if s.ExternalIds == nil {
		return nil
	}
	return *s.ExternalIds
}

func SubmissionLocationOf(s FormSubmission) *SubmissionLocation {
	return s.Location
}
