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
	Protocol *EmailProtocol `gorm:"size:12;"`
	Host     *string        `gorm:"size:255;"`
	Port     *uint          `gorm:"size:5;"`
	Username *string        `gorm:"size:255;"`
	Password *string        `gorm:"size:255;"`
}
