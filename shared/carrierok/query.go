package carrierok

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	UnitTypeTruck   = "TRUCK"
	UnitTypeTrailer = "TRAILER"

	SortOrderAsc  = "asc"
	SortOrderDesc = "desc"

	maxAutocompleteLimit  = 15
	minAutocompleteLength = 2
	maxMonitoringPageSize = 500
	monitoringDateLayout  = "20060102"
)

type ProfileQuery struct {
	DOTNumber    string
	DocketNumber string
	Company      string
	EIN          string
	Email        string
	Phone        string
	Address      string
	VIN          string
	PlateNumber  string
	PlateState   string
	UnitNumber   string
	UnitType     string
}

func (q *ProfileQuery) Validate() error {
	primaries := [...]struct {
		name  string
		value string
	}{
		{name: "dot_number", value: q.DOTNumber},
		{name: "docket_number", value: q.DocketNumber},
		{name: "company", value: q.Company},
		{name: "ein", value: q.EIN},
		{name: "email", value: q.Email},
		{name: "phone", value: q.Phone},
		{name: "address", value: q.Address},
		{name: "vin", value: q.VIN},
		{name: "plate_number", value: q.PlateNumber},
		{name: "unit_number", value: q.UnitNumber},
	}

	set := make([]string, 0, 2)
	for _, primary := range primaries {
		if strings.TrimSpace(primary.value) != "" {
			set = append(set, primary.name)
		}
	}
	switch len(set) {
	case 0:
		return fmt.Errorf("%w: exactly one identifier is required", ErrInvalidQuery)
	case 1:
	default:
		return fmt.Errorf(
			"%w: exactly one identifier is allowed, got %s",
			ErrInvalidQuery,
			strings.Join(set, ", "),
		)
	}

	if dot := strings.TrimSpace(q.DOTNumber); dot != "" && stringutils.DigitsOnly(dot) != dot {
		return fmt.Errorf("%w: dot_number must contain only digits", ErrInvalidQuery)
	}
	if strings.TrimSpace(q.PlateState) != "" && strings.TrimSpace(q.PlateNumber) == "" {
		return fmt.Errorf("%w: plate_state requires plate_number", ErrInvalidQuery)
	}
	if unitType := strings.TrimSpace(q.UnitType); unitType != "" {
		if strings.TrimSpace(q.UnitNumber) == "" {
			return fmt.Errorf("%w: unit_type requires unit_number", ErrInvalidQuery)
		}
		switch strings.ToUpper(unitType) {
		case UnitTypeTruck, UnitTypeTrailer:
		default:
			return fmt.Errorf("%w: unit_type must be TRUCK or TRAILER", ErrInvalidQuery)
		}
	}
	return nil
}

func (q *ProfileQuery) values() url.Values {
	values := make(url.Values, 2)
	setTrimmed(values, "dot_number", q.DOTNumber)
	setTrimmed(values, "docket_number", q.DocketNumber)
	setTrimmed(values, "company", q.Company)
	setTrimmed(values, "ein", q.EIN)
	setTrimmed(values, "email", q.Email)
	setTrimmed(values, "phone", q.Phone)
	setTrimmed(values, "address", q.Address)
	setTrimmed(values, "vin", q.VIN)
	setTrimmed(values, "plate_number", q.PlateNumber)
	setTrimmed(values, "plate_state", strings.ToUpper(q.PlateState))
	setTrimmed(values, "unit_number", q.UnitNumber)
	setTrimmed(values, "unit_type", strings.ToUpper(q.UnitType))
	return values
}

type FMCSAQuery struct {
	DOTNumber    string
	DocketNumber string
}

func (q FMCSAQuery) Validate() error {
	dot := strings.TrimSpace(q.DOTNumber)
	docket := strings.TrimSpace(q.DocketNumber)
	switch {
	case dot == "" && docket == "":
		return fmt.Errorf("%w: dot_number or docket_number is required", ErrInvalidQuery)
	case dot != "" && docket != "":
		return fmt.Errorf(
			"%w: only one of dot_number or docket_number is allowed",
			ErrInvalidQuery,
		)
	case dot != "" && stringutils.DigitsOnly(dot) != dot:
		return fmt.Errorf("%w: dot_number must contain only digits", ErrInvalidQuery)
	default:
		return nil
	}
}

func (q FMCSAQuery) values() url.Values {
	values := make(url.Values, 1)
	setTrimmed(values, "dot_number", q.DOTNumber)
	setTrimmed(values, "docket_number", q.DocketNumber)
	return values
}

type SearchParams struct {
	Query       string
	CompanyName string
	EIN         string
	VIN         string
	State       string
	Limit       int
	Offset      int
}

func (p *SearchParams) Validate() error {
	if strings.TrimSpace(p.Query) == "" &&
		strings.TrimSpace(p.CompanyName) == "" &&
		strings.TrimSpace(p.EIN) == "" &&
		strings.TrimSpace(p.VIN) == "" &&
		strings.TrimSpace(p.State) == "" {
		return fmt.Errorf("%w: at least one search criterion is required", ErrInvalidQuery)
	}
	if state := strings.TrimSpace(p.State); state != "" && utf8.RuneCountInString(state) != 2 {
		return fmt.Errorf("%w: state must be a two-letter code", ErrInvalidQuery)
	}
	if p.Limit < 0 {
		return fmt.Errorf("%w: limit must not be negative", ErrInvalidQuery)
	}
	if p.Offset < 0 {
		return fmt.Errorf("%w: offset must not be negative", ErrInvalidQuery)
	}
	return nil
}

func (p *SearchParams) values() url.Values {
	values := make(url.Values, 7)
	setTrimmed(values, "query", p.Query)
	setTrimmed(values, "company_name", p.CompanyName)
	setTrimmed(values, "ein", p.EIN)
	setTrimmed(values, "vin", p.VIN)
	setTrimmed(values, "state", strings.ToUpper(p.State))
	if p.Limit > 0 {
		values.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Offset > 0 {
		values.Set("offset", strconv.Itoa(p.Offset))
	}
	return values
}

func validateAutocomplete(q string, limit int) error {
	if utf8.RuneCountInString(strings.TrimSpace(q)) < minAutocompleteLength {
		return fmt.Errorf(
			"%w: autocomplete query must be at least %d characters",
			ErrInvalidQuery,
			minAutocompleteLength,
		)
	}
	if limit < 0 || limit > maxAutocompleteLimit {
		return fmt.Errorf(
			"%w: autocomplete limit must be between 0 and %d",
			ErrInvalidQuery,
			maxAutocompleteLimit,
		)
	}
	return nil
}

type MonitoringListParams struct {
	Page                 int
	PageSize             int
	ViewChanges          *bool
	ViewChangesInsurance *bool
	ViewChangesAuthority *bool
	ViewChangesSafety    *bool
	ViewChangesFleet     *bool
	ViewChangesContact   *bool
	ViewChangesRisk      *bool
	DateType             string
	DateMin              time.Time
	DateMax              time.Time
	SortBy               string
	SortOrder            string
	ProfileID            string
}

func (p *MonitoringListParams) Validate() error {
	if p.Page < 0 {
		return fmt.Errorf("%w: page must not be negative", ErrInvalidQuery)
	}
	if p.PageSize < 0 || p.PageSize > maxMonitoringPageSize {
		return fmt.Errorf(
			"%w: pageSize must be between 0 and %d",
			ErrInvalidQuery,
			maxMonitoringPageSize,
		)
	}
	if !p.DateMin.IsZero() && !p.DateMax.IsZero() && p.DateMin.After(p.DateMax) {
		return fmt.Errorf("%w: date_min must not be after date_max", ErrInvalidQuery)
	}
	if (!p.DateMin.IsZero() || !p.DateMax.IsZero()) && strings.TrimSpace(p.DateType) == "" {
		return fmt.Errorf("%w: date_type is required with a date range", ErrInvalidQuery)
	}
	switch strings.ToLower(strings.TrimSpace(p.SortOrder)) {
	case "", SortOrderAsc, SortOrderDesc:
	default:
		return fmt.Errorf("%w: sortOrder must be asc or desc", ErrInvalidQuery)
	}
	return nil
}

func (p *MonitoringListParams) values() url.Values {
	values := make(url.Values, 8)
	if p.Page > 0 {
		values.Set("page", strconv.Itoa(p.Page))
	}
	if p.PageSize > 0 {
		values.Set("pageSize", strconv.Itoa(p.PageSize))
	}
	setBool(values, "view_changes", p.ViewChanges)
	setBool(values, "view_changes_insurance", p.ViewChangesInsurance)
	setBool(values, "view_changes_authority", p.ViewChangesAuthority)
	setBool(values, "view_changes_safety", p.ViewChangesSafety)
	setBool(values, "view_changes_fleet", p.ViewChangesFleet)
	setBool(values, "view_changes_contact", p.ViewChangesContact)
	setBool(values, "view_changes_risk", p.ViewChangesRisk)
	setTrimmed(values, "date_type", p.DateType)
	if !p.DateMin.IsZero() {
		values.Set("date_min", p.DateMin.Format(monitoringDateLayout))
	}
	if !p.DateMax.IsZero() {
		values.Set("date_max", p.DateMax.Format(monitoringDateLayout))
	}
	setTrimmed(values, "sortBy", p.SortBy)
	setTrimmed(values, "sortOrder", strings.ToLower(p.SortOrder))
	setTrimmed(values, "profile_id", p.ProfileID)
	return values
}

func setTrimmed(values url.Values, key, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		values.Set(key, trimmed)
	}
}

func setBool(values url.Values, key string, value *bool) {
	if value != nil {
		values.Set(key, strconv.FormatBool(*value))
	}
}
