package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/workshop/restaurant-api/internal/model"
	"github.com/workshop/restaurant-api/internal/repository"
)

var ErrReservationNotFound = errors.New("reservation not found")

// Publisher publishes a raw JSON payload to a message broker.
type Publisher interface {
	Publish(ctx context.Context, payload []byte) error
}

type ReservationService interface {
	GetReservation(ctx context.Context, id int64) (*model.Reservation, error)
	ListReservations(ctx context.Context, restaurantID int64) ([]model.Reservation, error)
	BookReservation(ctx context.Context, r *model.Reservation) (int64, error)
	ConfirmReservation(ctx context.Context, id int64) error
	CancelReservation(ctx context.Context, id int64) error
}

type reservationService struct {
	repo      repository.ReservationRepository
	publisher Publisher
}

func NewReservationService(repo repository.ReservationRepository, publisher Publisher) ReservationService {
	return &reservationService{repo: repo, publisher: publisher}
}

func (s *reservationService) GetReservation(ctx context.Context, id int64) (*model.Reservation, error) {
	res, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrReservationNotFound
	}
	return res, nil
}

func (s *reservationService) ListReservations(ctx context.Context, restaurantID int64) ([]model.Reservation, error) {
	return s.repo.FindByRestaurant(ctx, restaurantID)
}

func (s *reservationService) BookReservation(ctx context.Context, r *model.Reservation) (int64, error) {
	if r.UserID <= 0 {
		return 0, errors.New("user_id is required")
	}
	if r.PartySize <= 0 {
		return 0, errors.New("party size must be greater than zero")
	}
	if r.ReservedForDate.IsZero() {
		return 0, errors.New("reserved_for_date is required")
	}

	r.Status = model.ReservationStatusWaiting

	id, err := s.repo.Create(ctx, r)
	if err != nil {
		return 0, err
	}
	r.ID = id

	if s.publisher != nil {
		payload, err := json.Marshal(r)
		if err != nil {
			slog.Error("failed to marshal reservation event", "reservation_id", id, "err", err)
		} else if err := s.publisher.Publish(ctx, payload); err != nil {
			slog.Error("failed to publish reservation event", "reservation_id", id, "err", err)
		} else {
			slog.Info("reservation event published", "reservation_id", id, "reservation_code", r.ReservationCode, "status", r.Status)
		}
	}

	return id, nil
}

func (s *reservationService) ConfirmReservation(ctx context.Context, id int64) error {
	res, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if res == nil {
		return ErrReservationNotFound
	}
	if res.Status != model.ReservationStatusWaiting {
		return errors.New("only waiting reservations can be confirmed")
	}
	return s.repo.UpdateStatus(ctx, id, model.ReservationStatusCalled)
}

func (s *reservationService) CancelReservation(ctx context.Context, id int64) error {
	res, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if res == nil {
		return ErrReservationNotFound
	}
	if res.Status == model.ReservationStatusSeated || res.Status == model.ReservationStatusCompleted {
		return errors.New("seated or completed reservations cannot be cancelled")
	}
	return s.repo.UpdateStatus(ctx, id, model.ReservationStatusCancelled)
}
