package drivers

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"
)

const (
	MaxListLimit                = 512
	ActivationStatusActive      = "active"
	ActivationStatusDeactivated = "deactivated"
	maxNameLength               = 255
	maxUsernameLength           = 189
)

type ListParams struct {
	DriverActivationStatus string
	Limit                  int
	After                  string
	ParentTagIDs           []string
	TagIDs                 []string
	AttributeValueIDs      []string
	Attributes             []string
	UpdatedAfterTime       *time.Time
	CreatedAfterTime       *time.Time
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p ListParams) Validate() error {
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > MaxListLimit) {
		return ErrListLimitInvalid
	}
	if p.DriverActivationStatus != "" &&
		!samsaraspec.ListDriversParamsDriverActivationStatus(p.DriverActivationStatus).Valid() {
		return fmt.Errorf("%w: %q", ErrDriverActivationStatusInvalid, p.DriverActivationStatus)
	}
	return nil
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p ListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "driverActivationStatus", p.DriverActivationStatus)
	httpx.SetInt(values, "limit", p.Limit)
	httpx.SetString(values, "after", p.After)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "attributeValueIds", p.AttributeValueIDs)
	httpx.SetTime(values, "updatedAfterTime", p.UpdatedAfterTime)
	httpx.SetTime(values, "createdAfterTime", p.CreatedAfterTime)
	for _, attr := range p.Attributes {
		if attr != "" {
			values.Add("attributes", attr)
		}
	}
	return values
}

//nolint:gocritic // request is copied intentionally to keep validation side-effect free.
func ValidateCreateRequest(req CreateRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ErrDriverNameRequired
	}
	if utf8.RuneCountInString(req.Name) > maxNameLength {
		return ErrDriverNameTooLong
	}
	if err := ValidateUsername(req.Username); err != nil {
		return err
	}
	if req.Password == "" {
		return ErrDriverPasswordRequired
	}
	return nil
}

func ValidateUsername(username string) error {
	if username == "" {
		return ErrDriverUsernameRequired
	}
	if utf8.RuneCountInString(username) > maxUsernameLength {
		return ErrDriverUsernameTooLong
	}
	if strings.ContainsFunc(username, func(r rune) bool {
		return r == '@' || unicode.IsSpace(r)
	}) {
		return ErrDriverUsernameInvalid
	}
	return nil
}
