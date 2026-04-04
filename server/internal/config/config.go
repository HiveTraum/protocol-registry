package config

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	GRPCPort    int    `envconfig:"GRPC_PORT" default:"50051"`
	HTTPPort    int    `envconfig:"HTTP_PORT" default:"8080"`
	PostgresDSN string `envconfig:"POSTGRES_DSN" required:"true"`
	S3Bucket    string `envconfig:"S3_BUCKET" required:"true"`
	S3Endpoint  string `envconfig:"S3_ENDPOINT" required:"true"`
	S3AccessKey string `envconfig:"S3_ACCESS_KEY" required:"true"`
	S3SecretKey string `envconfig:"S3_SECRET_KEY" required:"true"`
	S3Region    string `envconfig:"S3_REGION" default:"us-east-1"`
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
