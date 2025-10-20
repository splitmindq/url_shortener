package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"time"
)

type Config struct {
	Env         string `yaml:"env" env-required:"true"`
	StoragePath string `yaml:"storage_path" env-required:"true"`
	HttpServer  `yaml:"http_server"`
	App         `yaml:"app"`
	PostgresDB
}

type App struct {
	AliasLength int `yaml:"alias_length" env-required:"true"`
	MaxAttempts int `yaml:"max_attempts" env-required:"true"`
}

type HttpServer struct {
	Addr        string        `yaml:"address" env-required:"true"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type PostgresDB struct {
	Host     string `env:"PGHOST" env-required:"true"`
	Port     string `env:"PGPORT" env-required:"true"`
	User     string `env:"PGUSER" env-required:"true"`
	Password string `env:"PGPASSWORD" env-required:"true"`
	Database string `env:"PGDATABASE" env-required:"true"`
	ConnectionPoolConfig
}

type ConnectionPoolConfig struct {
	MaxConnections        int32         `yaml:"max_connections" env-default:"20"`
	MinConnections        int32         `yaml:"min_connections" env-default:"5"`
	MaxConnectionLifetime time.Duration `yaml:"max_connection_lifetime" env-default:"60m"`
	MaxConnectionIdleTime time.Duration `yaml:"max_connection_idle_time" env-default:"30m"`
}

func MustLoadConfig() *Config {
	//if err := godotenv.Load(".env"); err != nil {
	//	log.Fatal("Error loading .env file")
	//}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH environment variable not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatal("Config file does not exist")
	}

	cfg := new(Config)

	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		log.Fatal("Failed to read config:", err)
	}
	return cfg

}

func (p PostgresDB) ConnString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		p.Host, p.Port, p.User, p.Password, p.Database,
	)
}
