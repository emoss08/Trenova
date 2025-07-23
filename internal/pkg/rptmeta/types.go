// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package rptmeta

import (
	"github.com/emoss08/trenova/internal/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
)

// Caching configuration for the report
type Caching struct {
	IsCachable    bool `yaml:"isCachable"`    // Whether the report supports caching
	CacheDuration int  `yaml:"cacheDuration"` // Duration to cache the report (in seconds)
}

// Scheduling configuration for the report
type Scheduling struct {
	IsScheduled bool   `yaml:"isScheduled"`        // Whether the report is scheduled
	Schedule    string `yaml:"schedule,omitempty"` // Optional CRON expression for scheduling
}

// Metadata for the report itself
type Report struct {
	Title       string      `yaml:"title"`       // Title of the report
	Description string      `yaml:"description"` // Description of what the report does
	Tags        []string    `yaml:"tags"`        // Tags for easier filtering and searching
	Version     int         `yaml:"version"`     // Versioning for the report
	Caching     *Caching    `yaml:"caching"`     // Caching configuration
	Scheduling  *Scheduling `yaml:"scheduling"`  // Scheduling configuration
}

func (r *Report) Validate(multiErr *errors.MultiError) {
	err := validation.ValidateStruct(r,
		validation.Field(&r.Title, validation.Required.Error("Title is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Variables that define the report's parameters
type Variable struct {
	Name          string   `yaml:"name"`          // Name of the variable
	Placeholder   string   `yaml:"placeholder"`   // SQL placeholder (e.g., ":variable")
	Type          string   `yaml:"type"`          // Type of the variable (e.g., "string", "integer")
	Default       string   `yaml:"default"`       // Default value for the variable
	Description   string   `yaml:"description"`   // Description of the variable's purpose
	IsRequired    bool     `yaml:"isRequired"`    // Whether the variable is required
	AllowedValues []string `yaml:"allowedValues"` // Optional list of allowed values for validation
}

func (v *Variable) Validate(multiErr *errors.MultiError) {
	err := validation.ValidateStruct(v,
		validation.Field(&v.Name, validation.Required.Error("Name is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Full metadata for the report
type Metadata struct {
	Report    *Report     `yaml:"report"`               // Report metadata
	Variables []*Variable `yaml:"variables"`            // Variables used in the SQL
	SQL       string      `yaml:"sql"       json:"sql"` // SQL query
}

func (m *Metadata) Validate(multiErr *errors.MultiError) {
	err := validation.ValidateStruct(m,
		validation.Field(&m.Report, validation.Required.Error("Report is required")),
		validation.Field(&m.SQL, validation.Required.Error("SQL is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}
