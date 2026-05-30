package email

func (p *Profile) GetStaticFieldMap() map[string]string {
	return map[string]string{
		"id":             "id",
		"businessUnitId": "business_unit_id",
		"organizationId": "organization_id",
		"name":           "name",
		"description":    "description",
		"senderName":     "from_name",
		"senderEmail":    "from_address",
		"replyToEmail":   "reply_to",
		"provider":       "provider_type",
		"status":         "status",
		"version":        "version",
		"createdAt":      "created_at",
		"updatedAt":      "updated_at",
	}
}

func (m *Message) GetStaticFieldMap() map[string]string {
	return map[string]string{
		"id":                "id",
		"businessUnitId":    "business_unit_id",
		"organizationId":    "organization_id",
		"profileId":         "profile_id",
		"purpose":           "purpose",
		"provider":          "provider",
		"idempotencyKey":    "idempotency_key",
		"providerMessageId": "provider_message_id",
		"status":            "status",
		"attempts":          "attempts",
		"fromEmail":         "from_email",
		"fromName":          "from_name",
		"replyToEmail":      "reply_to_email",
		"subject":           "subject",
		"sentAt":            "sent_at",
		"deliveredAt":       "delivered_at",
		"failedAt":          "failed_at",
		"createdAt":         "created_at",
		"updatedAt":         "updated_at",
	}
}
