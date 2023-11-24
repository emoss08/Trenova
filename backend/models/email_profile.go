package models

type EmailProtocol string

const (
	TLS         EmailProtocol = "TLS"
	SSL         EmailProtocol = "SSL"
	UNENCRYPTED EmailProtocol = "UNENCRYPTED"
)

type EmailProfile struct {
	BaseModel
	Email    string
	Protocol *EmailProtocol `gorm:"size:12;type:email_protocol_type" json:"protocol" validate:"required,oneof=TLS SSL UNENCRYPTED,max=12"`
	Host     *string        `gorm:"size:255;" json:"host" validate:"omitempty,url"`
	Port     *uint          `gorm:"size:5;" json:"port" validate:"omitempty,max=5,number"`
	Username *string        `gorm:"size:255;" json:"username" validate:"omitempty"`
	Password *string        `gorm:"size:255;" json:"password" validate:"omitempty"`
}
