package pagedraft

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type Surface string

const (
	SurfaceShipmentImport = Surface("shipment_import")
	SurfaceFormula        = Surface("formula")
	SurfaceFormulaLibrary = Surface("formula_library")
)

func (s Surface) IsValid() bool {
	switch s {
	case SurfaceShipmentImport, SurfaceFormula, SurfaceFormulaLibrary:
		return true
	default:
		return false
	}
}

func AllSurfaces() []Surface {
	return []Surface{SurfaceShipmentImport, SurfaceFormula, SurfaceFormulaLibrary}
}

const (
	MaxImportFields       = 80
	MaxImportStops        = 20
	MaxFieldKeyLength     = 64
	MaxFieldLabelLength   = 120
	MaxFieldValueLength   = 1000
	MaxStopTextLength     = 300
	MaxSourceLength       = tarifftypes.MaxSourceLength
	MaxTemplateNameLength = 100
	MaxLibraryNameLength  = 64
	MaxDescriptionLength  = 2_000
)

type Draft struct {
	Surface        Surface         `json:"surface"`
	ShipmentImport *ShipmentImport `json:"shipmentImport,omitempty"`
	Formula        *Formula        `json:"formula,omitempty"`
	FormulaLibrary *FormulaLibrary `json:"formulaLibrary,omitempty"`
}

type FieldStatus string

const (
	FieldAccepted    = FieldStatus("accepted")
	FieldNeedsReview = FieldStatus("needs-review")
	FieldMissing     = FieldStatus("missing")
	FieldConflicting = FieldStatus("conflicting")
	FieldEdited      = FieldStatus("edited")
)

func (s FieldStatus) IsValid() bool {
	switch s {
	case FieldAccepted, FieldNeedsReview, FieldMissing, FieldConflicting, FieldEdited:
		return true
	default:
		return false
	}
}

type ShipmentImport struct {
	Fields   []ImportField  `json:"fields"`
	Required RequiredFields `json:"required"`
	Stops    []ImportStop   `json:"stops"`
}

type ImportField struct {
	Key        string      `json:"key"`
	Label      string      `json:"label"`
	Value      string      `json:"value"`
	Confidence float64     `json:"confidence"`
	Status     FieldStatus `json:"status"`
}

type RequiredField string

const (
	RequiredCustomer        = RequiredField("customerId")
	RequiredServiceType     = RequiredField("serviceTypeId")
	RequiredShipmentType    = RequiredField("shipmentTypeId")
	RequiredFormulaTemplate = RequiredField("formulaTemplateId")
)

func (f RequiredField) IsValid() bool {
	switch f {
	case RequiredCustomer, RequiredServiceType, RequiredShipmentType, RequiredFormulaTemplate:
		return true
	default:
		return false
	}
}

func AllRequiredFields() []RequiredField {
	return []RequiredField{
		RequiredCustomer,
		RequiredServiceType,
		RequiredShipmentType,
		RequiredFormulaTemplate,
	}
}

type RequiredFields struct {
	CustomerID        string `json:"customerId"`
	ServiceTypeID     string `json:"serviceTypeId"`
	ShipmentTypeID    string `json:"shipmentTypeId"`
	FormulaTemplateID string `json:"formulaTemplateId"`
}

func (r RequiredFields) Value(field RequiredField) string {
	switch field {
	case RequiredCustomer:
		return r.CustomerID
	case RequiredServiceType:
		return r.ServiceTypeID
	case RequiredShipmentType:
		return r.ShipmentTypeID
	case RequiredFormulaTemplate:
		return r.FormulaTemplateID
	default:
		return ""
	}
}

type StopRole string

const (
	StopPickup   = StopRole("pickup")
	StopDelivery = StopRole("delivery")
)

func (r StopRole) IsValid() bool {
	return r == StopPickup || r == StopDelivery
}

type ImportStop struct {
	Role         StopRole `json:"role"`
	Name         string   `json:"name"`
	AddressLine1 string   `json:"addressLine1"`
	City         string   `json:"city"`
	State        string   `json:"state"`
	PostalCode   string   `json:"postalCode"`
	Date         string   `json:"date"`
	TimeWindow   string   `json:"timeWindow"`
	LocationID   string   `json:"locationId"`
	Confidence   float64  `json:"confidence"`
}

func (s *ImportStop) HasLocation() bool {
	return strings.TrimSpace(s.LocationID) != ""
}

func (s *ImportStop) HasSchedule() bool {
	date := strings.TrimSpace(s.Date)
	if date == "" {
		return false
	}
	if seconds, err := strconv.ParseInt(date, 10, 64); err == nil {
		return seconds > 0
	}
	for _, layout := range scheduleLayouts {
		if _, err := time.Parse(layout, date); err == nil {
			return true
		}
	}

	return false
}

var scheduleLayouts = [...]string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04",
	"2006-01-02",
}

type Formula struct {
	TemplateID  string `json:"templateId,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
}

type FormulaLibrary struct {
	LibraryID   string `json:"libraryId,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
}

func (d *Draft) Validate(prefix string, multiErr *errortypes.MultiError) {
	field := fieldNamer(prefix)

	carried := map[Surface]bool{
		SurfaceShipmentImport: d.ShipmentImport != nil,
		SurfaceFormula:        d.Formula != nil,
		SurfaceFormulaLibrary: d.FormulaLibrary != nil,
	}
	if !d.Surface.IsValid() {
		multiErr.Add(field("surface"), errortypes.ErrInvalid, "Unknown draft surface")
	} else if !carried[d.Surface] {
		multiErr.Add(field(d.Surface.payloadField()), errortypes.ErrRequired,
			"A draft for this surface is required")
	}
	for _, surface := range AllSurfaces() {
		if surface != d.Surface && carried[surface] {
			multiErr.Add(field(surface.payloadField()), errortypes.ErrInvalid,
				"A draft carries only its own surface's payload")
		}
	}

	if d.ShipmentImport != nil {
		d.ShipmentImport.validate(field("shipmentImport"), multiErr)
	}
	if d.Formula != nil {
		d.Formula.validate(field("formula"), multiErr)
	}
	if d.FormulaLibrary != nil {
		d.FormulaLibrary.validate(field("formulaLibrary"), multiErr)
	}
}

func (s Surface) payloadField() string {
	switch s {
	case SurfaceFormula:
		return "formula"
	case SurfaceFormulaLibrary:
		return "formulaLibrary"
	case SurfaceShipmentImport:
		return "shipmentImport"
	}
	return "surface"
}

func (s *ShipmentImport) validate(prefix string, multiErr *errortypes.MultiError) {
	field := fieldNamer(prefix)

	if len(s.Fields) > MaxImportFields {
		multiErr.Add(field("fields"), errortypes.ErrInvalid, "Too many extracted fields")
	} else {
		seen := make(map[string]struct{}, len(s.Fields))
		for i := range s.Fields {
			item := &s.Fields[i]
			itemField := indexed(field("fields"), i)
			if !validKey(item.Key) {
				multiErr.Add(itemField+".key", errortypes.ErrInvalid, "Field key is invalid")
			} else if _, dup := seen[item.Key]; dup {
				multiErr.Add(itemField+".key", errortypes.ErrDuplicate, "Field key is repeated")
			}
			seen[item.Key] = struct{}{}
			checkLength(multiErr, itemField+".label", item.Label, MaxFieldLabelLength)
			checkLength(multiErr, itemField+".value", item.Value, MaxFieldValueLength)
			checkConfidence(multiErr, itemField+".confidence", item.Confidence)
			if !item.Status.IsValid() {
				multiErr.Add(itemField+".status", errortypes.ErrInvalid, "Field status is invalid")
			}
		}
	}

	for _, required := range AllRequiredFields() {
		checkOptionalID(
			multiErr,
			field("required")+"."+string(required),
			s.Required.Value(required),
		)
	}

	if len(s.Stops) > MaxImportStops {
		multiErr.Add(field("stops"), errortypes.ErrInvalid, "Too many stops")
		return
	}
	for i := range s.Stops {
		stop := &s.Stops[i]
		stopField := indexed(field("stops"), i)
		if !stop.Role.IsValid() {
			multiErr.Add(stopField+".role", errortypes.ErrInvalid, "Stop role is invalid")
		}
		for name, value := range map[string]string{
			"name":         stop.Name,
			"addressLine1": stop.AddressLine1,
			"city":         stop.City,
			"state":        stop.State,
			"postalCode":   stop.PostalCode,
			"date":         stop.Date,
			"timeWindow":   stop.TimeWindow,
		} {
			checkLength(multiErr, stopField+"."+name, value, MaxStopTextLength)
		}
		checkOptionalID(multiErr, stopField+".locationId", stop.LocationID)
		checkConfidence(multiErr, stopField+".confidence", stop.Confidence)
	}
}

func (f *Formula) validate(prefix string, multiErr *errortypes.MultiError) {
	field := fieldNamer(prefix)

	checkOptionalID(multiErr, field("templateId"), f.TemplateID)
	checkLength(multiErr, field("name"), f.Name, MaxTemplateNameLength)
	checkLength(multiErr, field("description"), f.Description, MaxDescriptionLength)
	checkSource(multiErr, field("source"), f.Source)
}

func (l *FormulaLibrary) validate(prefix string, multiErr *errortypes.MultiError) {
	field := fieldNamer(prefix)

	checkOptionalID(multiErr, field("libraryId"), l.LibraryID)
	checkLength(multiErr, field("name"), l.Name, MaxLibraryNameLength)
	checkLength(multiErr, field("description"), l.Description, MaxDescriptionLength)
	checkSource(multiErr, field("source"), l.Source)
}

func checkSource(multiErr *errortypes.MultiError, field, source string) {
	if len(source) > MaxSourceLength {
		multiErr.Add(field, errortypes.ErrInvalid, "Source cannot exceed 64 KiB")
	}
}

func (d *Draft) Normalized() *Draft {
	if d == nil {
		return nil
	}

	out := &Draft{Surface: Surface(strings.TrimSpace(string(d.Surface)))}
	if d.ShipmentImport != nil {
		out.ShipmentImport = d.ShipmentImport.normalized()
	}
	if d.Formula != nil {
		out.Formula = d.Formula.normalized()
	}
	if d.FormulaLibrary != nil {
		out.FormulaLibrary = d.FormulaLibrary.normalized()
	}

	return out
}

func (s *ShipmentImport) normalized() *ShipmentImport {
	out := &ShipmentImport{
		Fields: make([]ImportField, 0, len(s.Fields)),
		Required: RequiredFields{
			CustomerID:        strings.TrimSpace(s.Required.CustomerID),
			ServiceTypeID:     strings.TrimSpace(s.Required.ServiceTypeID),
			ShipmentTypeID:    strings.TrimSpace(s.Required.ShipmentTypeID),
			FormulaTemplateID: strings.TrimSpace(s.Required.FormulaTemplateID),
		},
		Stops: make([]ImportStop, 0, len(s.Stops)),
	}
	for _, item := range s.Fields {
		out.Fields = append(out.Fields, ImportField{
			Key:        strings.TrimSpace(item.Key),
			Label:      strings.TrimSpace(item.Label),
			Value:      strings.TrimSpace(item.Value),
			Confidence: item.Confidence,
			Status:     FieldStatus(strings.TrimSpace(string(item.Status))),
		})
	}
	for _, stop := range s.Stops {
		out.Stops = append(out.Stops, ImportStop{
			Role:         StopRole(strings.TrimSpace(string(stop.Role))),
			Name:         strings.TrimSpace(stop.Name),
			AddressLine1: strings.TrimSpace(stop.AddressLine1),
			City:         strings.TrimSpace(stop.City),
			State:        strings.TrimSpace(stop.State),
			PostalCode:   strings.TrimSpace(stop.PostalCode),
			Date:         strings.TrimSpace(stop.Date),
			TimeWindow:   strings.TrimSpace(stop.TimeWindow),
			LocationID:   strings.TrimSpace(stop.LocationID),
			Confidence:   stop.Confidence,
		})
	}

	return out
}

func (f *Formula) normalized() *Formula {
	return &Formula{
		TemplateID:  strings.TrimSpace(f.TemplateID),
		Name:        strings.TrimSpace(f.Name),
		Description: strings.TrimSpace(f.Description),
		Source:      f.Source,
	}
}

func (l *FormulaLibrary) normalized() *FormulaLibrary {
	return &FormulaLibrary{
		LibraryID:   strings.TrimSpace(l.LibraryID),
		Name:        strings.TrimSpace(l.Name),
		Description: strings.TrimSpace(l.Description),
		Source:      l.Source,
	}
}

func validKey(key string) bool {
	if key == "" || len(key) > MaxFieldKeyLength {
		return false
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '_', r == '-', r == '.':
		default:
			return false
		}
	}

	return true
}

func checkLength(multiErr *errortypes.MultiError, field, value string, limit int) {
	if utf8.RuneCountInString(value) > limit {
		multiErr.Add(field, errortypes.ErrInvalid, "Too long")
	}
}

func checkConfidence(multiErr *errortypes.MultiError, field string, confidence float64) {
	if math.IsNaN(confidence) || confidence < 0 || confidence > 1 {
		multiErr.Add(field, errortypes.ErrInvalid, "Confidence must be between 0 and 1")
	}
}

func checkOptionalID(multiErr *errortypes.MultiError, field, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if _, err := pulid.Parse(strings.TrimSpace(value)); err != nil {
		multiErr.Add(field, errortypes.ErrInvalid, "Identifier is invalid")
	}
}

func indexed(prefix string, index int) string {
	return prefix + "[" + strconv.Itoa(index) + "]"
}

func fieldNamer(prefix string) func(string) string {
	return func(name string) string {
		if prefix == "" {
			return name
		}

		return prefix + "." + name
	}
}
