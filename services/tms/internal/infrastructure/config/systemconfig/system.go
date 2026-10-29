package systemconfig

type Config struct {
	SystemUserPassword string `mapstructure:"systemUserPassword" validate:"required,min=1,max=72"`
}
