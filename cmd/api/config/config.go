package config

import (
	"github.com/caarlos0/env"
	// auto loads .env
	_ "github.com/joho/godotenv/autoload"
)

// Config for app
type Config struct {
	BotToken       string `env:"BOT_TOKEN"`
	Port           string `env:"PORT" envDefault:"4000"`
	DbName         string `env:"DB_NAME" envDefault:"order-bot-dev"`
	DbPassword     string `env:"DB_PASSWORD" envDefault:"postgres"`
	DbUser         string `env:"DB_USER" envDefault:"postgres"`
	DbHost         string `env:"DB_HOST" envDefault:"localhost"`
	RedisURL       string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`
	RedisPassword  string `env:"REDIS_PASSWORD" envDefault:"" json:"-"`
	RedisNamespace string `env:"REDIS_NAMESPACE" envDefault:"order_bot_dev"`
}

// New app config
func New() (Config, error) {
	cfg := Config{}
	err := env.Parse(&cfg)
	return cfg, err
}
