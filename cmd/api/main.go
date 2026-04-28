package main

import (
	"log/slog"

	"github.com/workshop/restaurant-api/internal/config"
	"github.com/workshop/restaurant-api/internal/repository"
	"github.com/workshop/restaurant-api/internal/server"
	"github.com/workshop/restaurant-api/internal/service"
	"github.com/workshop/restaurant-api/pkg/database"
	kafkapkg "github.com/workshop/restaurant-api/pkg/kafka"
)

func main() {
	cfg := config.Load()
	db := database.NewPostgres(cfg.DatabaseURL)
	defer db.Close()

	kafkaWriter := kafkapkg.NewWriter(cfg.KafkaBrokers, cfg.KafkaTopicReservation)
	defer kafkaWriter.Close()
	publisher := kafkapkg.NewKafkaPublisher(kafkaWriter)
	slog.Info("kafka producer ready", "brokers", cfg.KafkaBrokers, "topic", cfg.KafkaTopicReservation)

	reservationRepo := repository.NewReservationRepo(db)
	reservationSvc := service.NewReservationService(reservationRepo, publisher)

	restaurantRepo := repository.NewRestaurantRepo(db)
	restaurantSvc := service.NewRestaurantService(restaurantRepo, reservationRepo)

	slog.Info("starting server", "addr", cfg.HTTPAddr)
	if err := server.New(reservationSvc, restaurantSvc).Start(cfg.HTTPAddr); err != nil {
		slog.Error("server failed", "err", err)
	}
}
