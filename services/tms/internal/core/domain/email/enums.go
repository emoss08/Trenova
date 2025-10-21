package email

type ProviderType string

const (
	ProviderTypeSMTP    = ProviderType("SMTP")
	ProviderTypeResend  = ProviderType("Resend")
	ProviderTypeMailHog = ProviderType(
		"MailHog",
	) // * For testing purposes only or local development
)

type AuthType string

const (
	AuthTypeNone    = AuthType("None")
	AuthTypePlain   = AuthType("Plain")
	AuthTypeLogin   = AuthType("Login")
	AuthTypeCRAMMD5 = AuthType("CRAMMD5")
	AuthTypeOAuth2  = AuthType("OAuth2")
	AuthTypeAPIKey  = AuthType("APIKey")
)

type EncryptionType string

const (
	EncryptionTypeNone     = EncryptionType("None")
	EncryptionTypeSSLTLS   = EncryptionType("SSL_TLS")
	EncryptionTypeSTARTTLS = EncryptionType("StartTLS")
)

type TemplateCategory string

const (
	TemplateCategoryNotification = TemplateCategory("Notification")
	TemplateCategoryMarketing    = TemplateCategory("Marketing")
	TemplateCategorySystem       = TemplateCategory("System")
	TemplateCategoryCustom       = TemplateCategory("Custom")
)

type QueueStatus string

const (
	QueueStatusPending    = QueueStatus("Pending")
	QueueStatusProcessing = QueueStatus("Processing")
	QueueStatusSent       = QueueStatus("Sent")
	QueueStatusFailed     = QueueStatus("Failed")
	QueueStatusScheduled  = QueueStatus("Scheduled")
	QueueStatusCancelled  = QueueStatus("Cancelled")
)

type Priority string

const (
	PriorityHigh   = Priority("High")
	PriorityMedium = Priority("Medium")
	PriorityLow    = Priority("Low")
)

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

type TLSPolicy string

const (
	TLSPolicyMandatory     = TLSPolicy("Mandatory")
	TLSPolicyOpportunistic = TLSPolicy("Opportunistic")
	TLSPolicyNone          = TLSPolicy("None")
)

type BounceType string

const (
	BounceTypeHard  = BounceType("Hard")
	BounceTypeSoft  = BounceType("Soft")
	BounceTypeBlock = BounceType("Block")
)
