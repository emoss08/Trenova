package envs

import "github.com/spf13/viper"

type Config struct {
	Port   string `mapstructure:"PORT"`
	DB_DSN string `mapstructure:"DB_DSN"`
}

func LoadConfig() (c Config, err error) {
	viper.AddConfigPath("./pkg/common/config/envs")
	viper.SetConfigName("dev")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()

	if err != nil {
		return
	}

	err = viper.Unmarshal(&c)

	return
}
