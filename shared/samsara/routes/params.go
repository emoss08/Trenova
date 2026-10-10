package routes

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

const (
	maxListLimit               = 512
	minRouteStops              = 2
	maxNotesLength             = 2000
	maxStopAppointmentWindows  = 3
	IncludeStopsActualDistance = "stops.actualDistanceMeters"
)

type ListParams struct {
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	After        string
	Include      []string
	TagIDs       []string
	ParentTagIDs []string
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p ListParams) Validate() error {
	if p.StartTime == nil || p.EndTime == nil {
		return ErrTimeRangeRequired
	}
	if p.EndTime.Before(*p.StartTime) {
		return ErrTimeRangeInvalid
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > maxListLimit) {
		return ErrListLimitInvalid
	}
	return validateInclude(p.Include)
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p ListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetTime(values, "startTime", p.StartTime)
	httpx.SetTime(values, "endTime", p.EndTime)
	httpx.SetInt(values, "limit", p.Limit)
	httpx.SetString(values, "after", p.After)
	httpx.SetStringsCSV(values, "include", p.Include)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	return values
}

func validateInclude(include []string) error {
	for _, value := range include {
		if value != IncludeStopsActualDistance {
			return fmt.Errorf("%w: %q", ErrIncludeInvalid, value)
		}
	}
	return nil
}

//nolint:gocritic // request is copied intentionally to keep validation side-effect free.
func ValidateCreateRequest(req CreateRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return ErrRouteNameRequired
	}
	if err := validateNotes(req.Notes, ErrRouteNotesTooLong); err != nil {
		return err
	}
	if len(req.Stops) < minRouteStops {
		return ErrRouteStopsTooFew
	}
	for i := range req.Stops {
		stop := &req.Stops[i]
		if err := validateStop(stopFields{
			addressID:          stop.AddressId,
			singleUse:          stop.SingleUseLocation != nil,
			notes:              stop.Notes,
			appointmentWindows: countWindows(stop.AppointmentWindows),
		}); err != nil {
			return fmt.Errorf("stops[%d]: %w", i, err)
		}
	}
	return nil
}

func ValidateUpdateRequest(req UpdateRequest) error {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return ErrRouteNameRequired
	}
	if err := validateNotes(req.Notes, ErrRouteNotesTooLong); err != nil {
		return err
	}
	if req.Stops == nil {
		return nil
	}
	stops := *req.Stops
	for i := range stops {
		stop := &stops[i]
		fields := stopFields{
			addressID:          stop.AddressId,
			singleUse:          stop.SingleUseLocation != nil,
			notes:              stop.Notes,
			appointmentWindows: countWindows(stop.AppointmentWindows),
			existing:           stop.Id != nil && strings.TrimSpace(*stop.Id) != "",
		}
		if err := validateStop(fields); err != nil {
			return fmt.Errorf("stops[%d]: %w", i, err)
		}
	}
	return nil
}

type stopFields struct {
	addressID          *string
	singleUse          bool
	notes              *string
	appointmentWindows int
	existing           bool
}

func validateStop(fields stopFields) error {
	hasAddress := fields.addressID != nil && strings.TrimSpace(*fields.addressID) != ""
	if hasAddress && fields.singleUse {
		return ErrStopLocationRequired
	}
	if !fields.existing && !hasAddress && !fields.singleUse {
		return ErrStopLocationRequired
	}
	if fields.appointmentWindows > maxStopAppointmentWindows {
		return ErrStopAppointmentsTooMany
	}
	return validateNotes(fields.notes, ErrStopNotesTooLong)
}

func countWindows(windows *[]AppointmentWindow) int {
	if windows == nil {
		return 0
	}
	return len(*windows)
}

func validateNotes(notes *string, tooLong error) error {
	if notes != nil && utf8.RuneCountInString(*notes) > maxNotesLength {
		return tooLong
	}
	return nil
}
