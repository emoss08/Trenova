// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package integration

// FieldType represents the type of configuration field
type FieldType string

const (
	// FieldTypeText is a text input field
	FieldTypeText = FieldType("text")

	// FieldTypePassword is a password input field
	FieldTypePassword = FieldType("password")

	// FieldTypeSelect is a select dropdown field
	FieldTypeSelect = FieldType("select")

	// FieldTypeToggle is a toggle/checkbox field
	FieldTypeToggle = FieldType("toggle")

	// FieldTypeTextarea is a multiline text input field
	FieldTypeTextarea = FieldType("textarea")

	// FieldTypeEmail is an email input field
	FieldTypeEmail = FieldType("email")

	// FieldTypeNumber is a number input field
	FieldTypeNumber = FieldType("number")

	// FieldTypeUrl is a URL input field
	FieldTypeURL = FieldType("url")
)

// Field defines a configuration field for an integration
type Field struct {
	// Key is the field identifier used in code
	Key string `json:"key"`

	// Name is the human-readable name of the field
	Name string `json:"name"`

	// Description provides additional information about the field
	Description string `json:"description"`

	// Type defines the type of input field
	Type FieldType `json:"type"`

	// Required indicates if the field is required
	Required bool `json:"required"`

	// Default value for the field
	DefaultValue any `json:"defaultValue,omitempty"`

	// Options for select fields
	Options []FieldOption `json:"options,omitempty"`

	// Validation rules
	Validation *FieldValidation `json:"validation,omitempty"`

	// Placeholder text for input fields
	Placeholder string `json:"placeholder,omitempty"`

	// Group name if fields should be grouped
	Group string `json:"group,omitempty"`

	// Order specifies the display order
	Order int `json:"order"`
}

// FieldOption represents an option for select fields
type FieldOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// FieldValidation defines validation rules for a field
type FieldValidation struct {
	// Pattern is a regex pattern for validation
	Pattern string `json:"pattern,omitempty"`

	// Min value for numeric fields
	Min *float64 `json:"min,omitempty"`

	// Max value for numeric fields
	Max *float64 `json:"max,omitempty"`

	// MinLength for text fields
	MinLength *int `json:"minLength,omitempty"`

	// MaxLength for text fields
	MaxLength *int `json:"maxLength,omitempty"`

	// Custom validation message
	Message string `json:"message,omitempty"`
}

// TriggerEvent represents events that can trigger integration actions
type TriggerEvent string

const (
	// EventIssueCreated triggers when an issue is created
	EventIssueCreated = TriggerEvent("issue_created")

	// EventIssueUpdated triggers when an issue is updated
	EventIssueUpdated = TriggerEvent("issue_updated")

	// EventIssueCommented triggers when a comment is added to an issue
	EventIssueCommented = TriggerEvent("issue_commented")

	// EventIssueStatusChanged triggers when an issue's status changes
	EventIssueStatusChanged = TriggerEvent("issue_status_changed")

	// EventIssueAssigned triggers when an issue is assigned
	EventIssueAssigned = TriggerEvent("issue_assigned")
)

// EventTrigger defines a trigger that can initiate integration actions
type EventTrigger struct {
	// Event is the type of event that triggers the action
	Event TriggerEvent `json:"event"`

	// Description explains what happens when triggered
	Description string `json:"description"`

	// Enabled indicates if the trigger is active
	Enabled bool `json:"enabled"`

	// RequiredFields are fields that must be configured for this trigger
	RequiredFields []string `json:"requiredFields,omitempty"`
}

// WebhookEndpoint defines an endpoint for integration webhooks
type WebhookEndpoint struct {
	// Name of the webhook endpoint
	Name string `json:"name"`

	// URL of the webhook endpoint
	URL string `json:"url"`

	// Description explains the purpose of this webhook
	Description string `json:"description"`

	// Enabled indicates if the webhook is active
	Enabled bool `json:"enabled"`

	// Secret for webhook authentication
	Secret string `json:"secret,omitempty"`

	// Events that trigger this webhook
	Events []TriggerEvent `json:"events"`
}
