package email

// ProviderType represents the type of email provider
type ProviderType string

const (
	ProviderTypeSMTP      = ProviderType("SMTP")
	ProviderTypeSendGrid  = ProviderType("SendGrid")
	ProviderTypeAWSSES    = ProviderType("AWS_SES")
	ProviderTypeMailgun   = ProviderType("Mailgun")
	ProviderTypePostmark  = ProviderType("Postmark")
	ProviderTypeExchange  = ProviderType("Exchange")
	ProviderTypeOffice365 = ProviderType("Office365")
)

// AuthType represents the authentication method for email providers
type AuthType string

const (
	AuthTypePlain   = AuthType("Plain")
	AuthTypeLogin   = AuthType("Login")
	AuthTypeCRAMMD5 = AuthType("CRAMMD5")
	AuthTypeOAuth2  = AuthType("OAuth2")
	AuthTypeAPIKey  = AuthType("APIKey")
)

// EncryptionType represents the encryption method for email connections
type EncryptionType string

const (
	EncryptionTypeNone     = EncryptionType("None")
	EncryptionTypeSSLTLS   = EncryptionType("SSL_TLS")
	EncryptionTypeSTARTTLS = EncryptionType("StartTLS")
)

// TemplateCategory represents the category of an email template
type TemplateCategory string

const (
	TemplateCategoryNotification = TemplateCategory("Notification")
	TemplateCategoryMarketing    = TemplateCategory("Marketing")
	TemplateCategorySystem       = TemplateCategory("System")
	TemplateCategoryCustom       = TemplateCategory("Custom")
)

// QueueStatus represents the status of an email in the queue
type QueueStatus string

const (
	QueueStatusPending    = QueueStatus("Pending")
	QueueStatusProcessing = QueueStatus("Processing")
	QueueStatusSent       = QueueStatus("Sent")
	QueueStatusFailed     = QueueStatus("Failed")
	QueueStatusScheduled  = QueueStatus("Scheduled")
	QueueStatusCancelled  = QueueStatus("Cancelled")
)

// Priority represents the priority level of an email
type Priority string

const (
	PriorityHigh   = Priority("High")
	PriorityMedium = Priority("Medium")
	PriorityLow    = Priority("Low")
)

// LogStatus represents the delivery status in email logs
type LogStatus string

const (
	LogStatusDelivered    = LogStatus("Delivered")
	LogStatusOpened       = LogStatus("Opened")
	LogStatusClicked      = LogStatus("Clicked")
	LogStatusBounced      = LogStatus("Bounced")
	LogStatusComplained   = LogStatus("Complained")
	LogStatusUnsubscribed = LogStatus("Unsubscribed")
	LogStatusRejected     = LogStatus("Rejected")
)

// BounceType represents the type of email bounce
type BounceType string

const (
	BounceTypeHard  = BounceType("Hard")
	BounceTypeSoft  = BounceType("Soft")
	BounceTypeBlock = BounceType("Block")
)
