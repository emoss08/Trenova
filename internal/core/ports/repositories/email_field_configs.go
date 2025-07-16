package repositories

import "github.com/emoss08/trenova/internal/core/ports"

// EmailTemplateFieldConfig defines filterable and sortable fields for email templates
var EmailTemplateFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"name":        true,
		"slug":        true,
		"status":      true,
		"type":        true,
		"isProtected": true,
	},
	SortableFields: map[string]bool{
		"name":      true,
		"slug":      true,
		"status":    true,
		"type":      true,
		"createdAt": true,
		"updatedAt": true,
	},
	FieldMap: map[string]string{
		"name":        "name",
		"slug":        "slug",
		"status":      "status",
		"type":        "type",
		"isProtected": "is_protected",
		"createdAt":   "created_at",
		"updatedAt":   "updated_at",
	},
	EnumMap: map[string]bool{
		"status": true,
		"type":   true,
	},
	NestedFields: map[string]ports.NestedFieldDefinition{},
}

// EmailQueueFieldConfig defines filterable and sortable fields for email queue
var EmailQueueFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"status":       true,
		"priority":     true,
		"profileId":    true,
		"sentAt":       true,
		"scheduledAt":  true,
		"errorMessage": true,
	},
	SortableFields: map[string]bool{
		"status":      true,
		"priority":    true,
		"sentAt":      true,
		"scheduledAt": true,
		"createdAt":   true,
		"updatedAt":   true,
	},
	FieldMap: map[string]string{
		"status":       "status",
		"priority":     "priority",
		"profileId":    "profile_id",
		"sentAt":       "sent_at",
		"scheduledAt":  "scheduled_at",
		"errorMessage": "error_message",
		"createdAt":    "created_at",
		"updatedAt":    "updated_at",
	},
	EnumMap: map[string]bool{
		"status":   true,
		"priority": true,
	},
	NestedFields: map[string]ports.NestedFieldDefinition{},
}

// EmailLogFieldConfig defines filterable and sortable fields for email logs
var EmailLogFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"queueId":          true,
		"messageId":        true,
		"status":           true,
		"providerResponse": true,
		"openedAt":         true,
		"clickedAt":        true,
		"bouncedAt":        true,
	},
	SortableFields: map[string]bool{
		"status":    true,
		"openedAt":  true,
		"clickedAt": true,
		"bouncedAt": true,
		"createdAt": true,
	},
	FieldMap: map[string]string{
		"queueId":          "queue_id",
		"messageId":        "message_id",
		"status":           "status",
		"providerResponse": "provider_response",
		"openedAt":         "opened_at",
		"clickedAt":        "clicked_at",
		"bouncedAt":        "bounced_at",
		"createdAt":        "created_at",
	},
	EnumMap: map[string]bool{
		"status": true,
	},
	NestedFields: map[string]ports.NestedFieldDefinition{},
}
