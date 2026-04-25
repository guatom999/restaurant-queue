package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/workshop/restaurant-api/internal/config"
	"github.com/workshop/restaurant-api/internal/repository"
	"github.com/workshop/restaurant-api/internal/service"
	"github.com/workshop/restaurant-api/internal/worker"
	"github.com/workshop/restaurant-api/pkg/database"
	kafkapkg "github.com/workshop/restaurant-api/pkg/kafka"
)

func main() {
	cfg := config.Load()
	db := database.NewPostgres(cfg.DatabaseURL)
	defer db.Close()

	reservationRepo := repository.NewReservationRepo(db)
	reservationSvc := service.NewReservationService(reservationRepo)

	reader := kafkapkg.NewReader(cfg.KafkaBrokers, cfg.KafkaTopicReservation, "reservation-worker-group")
	defer reader.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("worker starting", "topic", cfg.KafkaTopicReservation)
	worker.NewReservationWorker(reader, reservationSvc).Run(ctx)
}
