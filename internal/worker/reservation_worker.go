package worker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/workshop/restaurant-api/internal/model"
	"github.com/workshop/restaurant-api/internal/service"
)

type ReservationWorker struct {
	reader *kafka.Reader
	svc    service.ReservationService
}

func NewReservationWorker(reader *kafka.Reader, svc service.ReservationService) *ReservationWorker {
	return &ReservationWorker{reader: reader, svc: svc}
}

func (w *ReservationWorker) Run(ctx context.Context) {
	slog.Info("reservation worker started")
	for {
		msg, err := w.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("reservation worker stopped")
				return
			}
			slog.Error("failed to read message", "err", err)
			continue
		}
		if err := w.process(ctx, msg); err != nil {
			slog.Error("failed to process message", "offset", msg.Offset, "err", err)
		}
	}
}

func (w *ReservationWorker) process(ctx context.Context, msg kafka.Message) error {
	var res model.Reservation
	if err := json.Unmarshal(msg.Value, &res); err != nil {
		return err
	}

	slog.Info("processing reservation event", "id", res.ID, "status", res.Status)

	switch res.Status {
	case model.ReservationStatusWaiting:
		return w.svc.ConfirmReservation(ctx, res.ID)
	default:
		slog.Info("no action for reservation status", "status", res.Status)
	}
	return nil
}
