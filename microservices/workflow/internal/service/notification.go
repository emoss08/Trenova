package service

type NotificationRequest struct {
	Type       string `json:"type"`
	EntityID   string `json:"entityId"`
	EntityType string `json:"entityType"`
	TenantID   string `json:"tenantId"`
	Subject    string `json:"subject"`
	Message    string `json:"message"`
}
