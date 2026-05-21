package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Port               string `mapstructure:"PORT"`
	DBHost             string `mapstructure:"DB_HOST"`
	DBUser             string `mapstructure:"DB_USER"`
	DBPassword         string `mapstructure:"DB_PASSWORD"`
	DBName             string `mapstructure:"DB_NAME"`
	DBPort             string `mapstructure:"DB_PORT"`
	RedisURL           string `mapstructure:"REDIS_URL"`
	JWTSecret          string `mapstructure:"JWT_SECRET"`
	PrivyWebhookSecret string `mapstructure:"PRIVY_WEBHOOK_SECRET"`
	XenditAPIKey       string `mapstructure:"XENDIT_API_KEY"`
	XenditWebhookToken string `mapstructure:"XENDIT_WEBHOOK_TOKEN"`
	BiteshipAPIKey     string `mapstructure:"BITESHIP_API_KEY"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		log.Println("No .env file found or error reading it, relying on system environment variables")
	}

	var cfg Config
	err = viper.Unmarshal(&cfg)
	return &cfg, err
}
