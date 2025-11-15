package workflow

// ScheduledTriggerConfig holds configuration for scheduled triggers
type ScheduledTriggerConfig struct {
	CronExpression string `json:"cronExpression"`
	Timezone       string `json:"timezone,omitempty"` // Defaults to UTC
}

// ShipmentStatusTriggerConfig holds configuration for shipment status change triggers
type ShipmentStatusTriggerConfig struct {
	Statuses []string `json:"statuses"` // List of statuses that trigger the workflow
}

// DocumentUploadTriggerConfig holds configuration for document upload triggers
type DocumentUploadTriggerConfig struct {
	DocumentTypes []string `json:"documentTypes"` // List of document types that trigger the workflow
	EntityTypes   []string `json:"entityTypes,omitempty"` // Optional: Filter by entity type (shipment, customer, etc.)
}

// EntityEventTriggerConfig holds configuration for entity create/update triggers
type EntityEventTriggerConfig struct {
	EntityType string `json:"entityType"` // Type of entity (shipment, customer, etc.)
}

// WebhookTriggerConfig holds configuration for webhook triggers
type WebhookTriggerConfig struct {
	WebhookURL    string            `json:"webhookUrl,omitempty"` // Optional: For outbound webhooks
	RequireAuth   bool              `json:"requireAuth"`
	AuthToken     string            `json:"authToken,omitempty"` // Optional: For authentication
	CustomHeaders map[string]string `json:"customHeaders,omitempty"`
}

// ManualTriggerConfig holds configuration for manual triggers
type ManualTriggerConfig struct {
	RequireConfirmation bool   `json:"requireConfirmation"`
	ConfirmationMessage string `json:"confirmationMessage,omitempty"`
}
