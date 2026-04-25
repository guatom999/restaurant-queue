package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr              string
	DatabaseURL           string
	KafkaBrokers          []string
	KafkaTopicReservation string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}
	return &Config{
		HTTPAddr:              getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/restaurant_db?sslmode=disable"),
		KafkaBrokers:          splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopicReservation: getEnv("KAFKA_TOPIC_RESERVATION", "reservation.requested"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	return strings.Split(s, ",")
}
