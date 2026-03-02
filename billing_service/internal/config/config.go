package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDSN string

	KafkaBrokers []string

	OrderCreatedTopic   string
	OrderPaidTopic      string
	OrderCancelledTopic string
}

func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %q is not set", key)
	}
	return v
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		mustGetenv("DB_HOST"),
		mustGetenv("DB_USER"),
		mustGetenv("DB_PASSWORD"),
		mustGetenv("DB_NAME"),
		mustGetenv("DB_PORT"),
		sslMode,
	)

	brokersRaw := mustGetenv("KAFKA_BROKERS")
	brokers := strings.Split(brokersRaw, ",")

	return &Config{
		DBDSN:               dsn,
		KafkaBrokers:        brokers,
		OrderCreatedTopic:   mustGetenv("ORDER_CREATED_TOPIC"),
		OrderPaidTopic:      mustGetenv("ORDER_PAID_TOPIC"),
		OrderCancelledTopic: mustGetenv("ORDER_CANCELLED_TOPIC"),
	}
}
