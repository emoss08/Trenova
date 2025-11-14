package report

import (
	"database/sql/driver"
	"fmt"
)

type Format string

const (
	FormatCSV   Format = "Csv"
	FormatExcel Format = "Excel"
)

func (f Format) String() string {
	return string(f)
}

func (f *Format) Scan(value any) error {
	if value == nil {
		*f = ""
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan Format: unexpected type %T", value)
	}

	*f = Format(str)
	return nil
}

func (f Format) Value() (driver.Value, error) {
	if f == "" {
		return nil, nil
	}
	return string(f), nil
}

func FormatFromString(s string) (Format, error) {
	switch s {
	case "Csv":
		return FormatCSV, nil
	case "Excel":
		return FormatExcel, nil
	default:
		return "", fmt.Errorf("invalid format: %s", s)
	}
}

func (f Format) IsValid() bool {
	switch f {
	case FormatCSV, FormatExcel:
		return true
	default:
		return false
	}
}

type DeliveryMethod string

const (
	DeliveryMethodDownload DeliveryMethod = "Download"
	DeliveryMethodEmail    DeliveryMethod = "Email"
)

func (d DeliveryMethod) String() string {
	return string(d)
}

func (d *DeliveryMethod) Scan(value any) error {
	if value == nil {
		*d = ""
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan DeliveryMethod: unexpected type %T", value)
	}

	*d = DeliveryMethod(str)
	return nil
}

func (d DeliveryMethod) Value() (driver.Value, error) {
	if d == "" {
		return nil, nil
	}
	return string(d), nil
}

func DeliveryMethodFromString(s string) (DeliveryMethod, error) {
	switch s {
	case "Download":
		return DeliveryMethodDownload, nil
	case "Email":
		return DeliveryMethodEmail, nil
	default:
		return "", fmt.Errorf("invalid delivery method: %s", s)
	}
}

func (d DeliveryMethod) IsValid() bool {
	switch d {
	case DeliveryMethodDownload, DeliveryMethodEmail:
		return true
	default:
		return false
	}
}

type Status string

const (
	StatusPending    Status = "Pending"
	StatusProcessing Status = "Processing"
	StatusCompleted  Status = "Completed"
	StatusFailed     Status = "Failed"
)

func (s Status) String() string {
	return string(s)
}

func (s *Status) Scan(value any) error {
	if value == nil {
		*s = ""
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan Status: unexpected type %T", value)
	}

	*s = Status(str)
	return nil
}

func (s Status) Value() (driver.Value, error) {
	if s == "" {
		return nil, nil
	}
	return string(s), nil
}

func StatusFromString(str string) (Status, error) {
	switch str {
	case "Pending":
		return StatusPending, nil
	case "Processing":
		return StatusProcessing, nil
	case "Completed":
		return StatusCompleted, nil
	case "Failed":
		return StatusFailed, nil
	default:
		return "", fmt.Errorf("invalid status: %s", str)
	}
}

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusProcessing, StatusCompleted, StatusFailed:
		return true
	default:
		return false
	}
}
