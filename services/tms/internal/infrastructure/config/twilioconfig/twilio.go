package twilioconfig

type Config struct {
	Enabled    bool   `mapstructure:"enabled"`
	AccountSID string `mapstructure:"accountSID" validate:"required_if=Enabled true"`
	AuthToken  string `mapstructure:"authToken"  validate:"required_if=Enabled true"`
	FromNumber string `mapstructure:"fromNumber" validate:"required_if=Enabled true"`
}
