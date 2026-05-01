package worker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
	"github.com/workshop/restaurant-api/internal/model"
)

type ReservationWorker struct {
	reader *kafka.Reader
}

func NewReservationWorker(reader *kafka.Reader) *ReservationWorker {
	return &ReservationWorker{reader: reader}
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

func (w *ReservationWorker) process(_ context.Context, msg kafka.Message) error {
	var res model.Reservation
	if err := json.Unmarshal(msg.Value, &res); err != nil {
		return err
	}

	slog.Info("new reservation awaiting staff confirmation",
		"reservation_id", res.ID,
		"reservation_code", res.ReservationCode,
		"restaurant_id", res.RestaurantID,
		"party_size", res.PartySize,
		"reserved_for_date", res.ReservedForDate,
		"status", res.Status,
	)

	// TODO: send booking confirmation notification to customer
	// TODO: send new queue alert to restaurant staff

	return nil
}
