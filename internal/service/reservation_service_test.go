package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/workshop/restaurant-api/internal/model"
)

// ─── mocks ───────────────────────────────────────────────────────────────────

type mockReservationRepo struct {
	findByIDFn               func(ctx context.Context, id int64) (*model.Reservation, error)
	findByRestaurantFn       func(ctx context.Context, restaurantID int64) ([]model.Reservation, error)
	bookFn                   func(ctx context.Context, r *model.Reservation) (int64, error)
	updateStatusFn           func(ctx context.Context, id int64, status model.ReservationStatus) error
	countActiveByDateFn      func(ctx context.Context, restaurantID int64, date time.Time) (int, error)
	findAvailabilityByDateFn func(ctx context.Context, restaurantID int64, date time.Time) ([]model.Reservation, error)
}

func (m *mockReservationRepo) FindByID(ctx context.Context, id int64) (*model.Reservation, error) {
	return m.findByIDFn(ctx, id)
}
func (m *mockReservationRepo) FindByRestaurant(ctx context.Context, restaurantID int64) ([]model.Reservation, error) {
	return m.findByRestaurantFn(ctx, restaurantID)
}
func (m *mockReservationRepo) Book(ctx context.Context, r *model.Reservation) (int64, error) {
	return m.bookFn(ctx, r)
}
func (m *mockReservationRepo) UpdateStatus(ctx context.Context, id int64, status model.ReservationStatus) error {
	return m.updateStatusFn(ctx, id, status)
}
func (m *mockReservationRepo) CountActiveByDate(ctx context.Context, restaurantID int64, date time.Time) (int, error) {
	return m.countActiveByDateFn(ctx, restaurantID, date)
}
func (m *mockReservationRepo) FindAvailabilityByDate(ctx context.Context, restaurantID int64, date time.Time) ([]model.Reservation, error) {
	if m.findAvailabilityByDateFn != nil {
		return m.findAvailabilityByDateFn(ctx, restaurantID, date)
	}
	return nil, nil
}

type mockPublisher struct {
	publishFn func(ctx context.Context, payload []byte) error
}

func (m *mockPublisher) Publish(ctx context.Context, payload []byte) error {
	return m.publishFn(ctx, payload)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func validReservation() *model.Reservation {
	return &model.Reservation{
		UserID:          1,
		RestaurantID:    10,
		PartySize:       2,
		ReservedForDate: time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC),
	}
}

func waitingReservation(id int64) *model.Reservation {
	return &model.Reservation{ID: id, Status: model.ReservationStatusWaiting}
}

// ─── GetReservation ───────────────────────────────────────────────────────────

func TestGetReservation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		findByID  func(context.Context, int64) (*model.Reservation, error)
		wantNil   bool
		wantErrIs error
		wantErr   bool
	}{
		{
			name: "returns reservation",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return &model.Reservation{ID: 1}, nil
			},
		},
		{
			name: "not found returns sentinel",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return nil, nil
			},
			wantNil:   true,
			wantErrIs: ErrReservationNotFound,
		},
		{
			name: "repo error propagates",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return nil, errors.New("db error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewReservationService(&mockReservationRepo{findByIDFn: tt.findByID}, nil)
			got, err := svc.GetReservation(ctx, 1)

			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && got == nil {
				t.Error("expected reservation, got nil")
			}
		})
	}
}

// ─── ListReservations ─────────────────────────────────────────────────────────

func TestListReservations(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates to repo", func(t *testing.T) {
		want := []model.Reservation{{ID: 1}, {ID: 2}}
		repo := &mockReservationRepo{
			findByRestaurantFn: func(_ context.Context, _ int64) ([]model.Reservation, error) {
				return want, nil
			},
		}
		svc := NewReservationService(repo, nil)
		got, err := svc.ListReservations(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Errorf("expected %d, got %d", len(want), len(got))
		}
	})

	t.Run("repo error propagates", func(t *testing.T) {
		repo := &mockReservationRepo{
			findByRestaurantFn: func(_ context.Context, _ int64) ([]model.Reservation, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewReservationService(repo, nil)
		_, err := svc.ListReservations(ctx, 10)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

// ─── BookReservation ──────────────────────────────────────────────────────────

func TestBookReservation_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	repo := &mockReservationRepo{}
	svc := NewReservationService(repo, nil)

	tests := []struct {
		name        string
		reservation *model.Reservation
		wantMsg     string
	}{
		{
			name:        "missing user_id",
			reservation: &model.Reservation{PartySize: 2, ReservedForDate: time.Now()},
			wantMsg:     "user_id is required",
		},
		{
			name:        "party size zero",
			reservation: &model.Reservation{UserID: 1, PartySize: 0, ReservedForDate: time.Now()},
			wantMsg:     "party size must be greater than zero",
		},
		{
			name:        "missing reserved_for_date",
			reservation: &model.Reservation{UserID: 1, PartySize: 2},
			wantMsg:     "reserved_for_date is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.BookReservation(ctx, tt.reservation)
			if err == nil || err.Error() != tt.wantMsg {
				t.Errorf("expected error %q, got %v", tt.wantMsg, err)
			}
		})
	}
}

func TestBookReservation_RepoErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("book error propagates", func(t *testing.T) {
		repo := &mockReservationRepo{
			bookFn: func(_ context.Context, _ *model.Reservation) (int64, error) {
				return 0, errors.New("db error")
			},
		}
		svc := NewReservationService(repo, nil)
		_, err := svc.BookReservation(ctx, validReservation())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestBookReservation_Success(t *testing.T) {
	ctx := context.Background()

	successRepo := func() *mockReservationRepo {
		return &mockReservationRepo{
			bookFn: func(_ context.Context, r *model.Reservation) (int64, error) {
				r.QueueNumber = 3
				r.ReservationCode = "R10-20260507-Q0003"
				return 42, nil
			},
		}
	}

	t.Run("without publisher", func(t *testing.T) {
		svc := NewReservationService(successRepo(), nil)
		id, err := svc.BookReservation(ctx, validReservation())
		if err != nil {
			t.Fatal(err)
		}
		if id != 42 {
			t.Errorf("expected id 42, got %d", id)
		}
	})

	t.Run("with publisher success", func(t *testing.T) {
		published := false
		pub := &mockPublisher{
			publishFn: func(_ context.Context, _ []byte) error {
				published = true
				return nil
			},
		}
		svc := NewReservationService(successRepo(), pub)
		r := validReservation()
		id, err := svc.BookReservation(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		if id != 42 {
			t.Errorf("expected id 42, got %d", id)
		}
		if !published {
			t.Error("expected publisher to be called")
		}
	})

	t.Run("publisher error does not fail booking", func(t *testing.T) {
		pub := &mockPublisher{
			publishFn: func(_ context.Context, _ []byte) error {
				return errors.New("kafka down")
			},
		}
		svc := NewReservationService(successRepo(), pub)
		id, err := svc.BookReservation(ctx, validReservation())
		if err != nil {
			t.Errorf("publish error should not propagate, got: %v", err)
		}
		if id != 42 {
			t.Errorf("expected id 42, got %d", id)
		}
	})

	t.Run("reservation code and queue number set correctly", func(t *testing.T) {
		svc := NewReservationService(successRepo(), nil)
		r := validReservation()
		_, err := svc.BookReservation(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		if r.QueueNumber != 3 {
			t.Errorf("expected queue number 3, got %d", r.QueueNumber)
		}
		if r.ReservationCode != "R10-20260507-Q0003" {
			t.Errorf("expected code R10-20260507-Q0003, got %s", r.ReservationCode)
		}
		if r.Status != model.ReservationStatusWaiting {
			t.Errorf("expected status WAITING, got %s", r.Status)
		}
	})
}

// ─── ConfirmReservation ───────────────────────────────────────────────────────

func TestConfirmReservation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		findByID  func(context.Context, int64) (*model.Reservation, error)
		updateFn  func(context.Context, int64, model.ReservationStatus) error
		wantErrIs error
		wantErr   bool
	}{
		{
			name: "success",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return waitingReservation(1), nil
			},
			updateFn: func(_ context.Context, _ int64, _ model.ReservationStatus) error {
				return nil
			},
		},
		{
			name: "not found",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return nil, nil
			},
			wantErrIs: ErrReservationNotFound,
		},
		{
			name: "repo find error",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return nil, errors.New("db error")
			},
			wantErr: true,
		},
		{
			name: "already called cannot be confirmed",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return &model.Reservation{ID: 1, Status: model.ReservationStatusCalled}, nil
			},
			wantErr: true,
		},
		{
			name: "update status error",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return waitingReservation(1), nil
			},
			updateFn: func(_ context.Context, _ int64, _ model.ReservationStatus) error {
				return errors.New("update error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockReservationRepo{
				findByIDFn:     tt.findByID,
				updateStatusFn: tt.updateFn,
			}
			svc := NewReservationService(repo, nil)
			err := svc.ConfirmReservation(ctx, 1)

			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ─── CancelReservation ────────────────────────────────────────────────────────

func TestCancelReservation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		findByID  func(context.Context, int64) (*model.Reservation, error)
		updateFn  func(context.Context, int64, model.ReservationStatus) error
		wantErrIs error
		wantErr   bool
	}{
		{
			name: "cancel waiting reservation",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return waitingReservation(1), nil
			},
			updateFn: func(_ context.Context, _ int64, _ model.ReservationStatus) error {
				return nil
			},
		},
		{
			name: "cancel called reservation",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return &model.Reservation{ID: 1, Status: model.ReservationStatusCalled}, nil
			},
			updateFn: func(_ context.Context, _ int64, _ model.ReservationStatus) error {
				return nil
			},
		},
		{
			name: "not found",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return nil, nil
			},
			wantErrIs: ErrReservationNotFound,
		},
		{
			name: "repo find error",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return nil, errors.New("db error")
			},
			wantErr: true,
		},
		{
			name: "seated cannot be cancelled",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return &model.Reservation{ID: 1, Status: model.ReservationStatusSeated}, nil
			},
			wantErr: true,
		},
		{
			name: "completed cannot be cancelled",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return &model.Reservation{ID: 1, Status: model.ReservationStatusCompleted}, nil
			},
			wantErr: true,
		},
		{
			name: "update status error",
			findByID: func(_ context.Context, _ int64) (*model.Reservation, error) {
				return waitingReservation(1), nil
			},
			updateFn: func(_ context.Context, _ int64, _ model.ReservationStatus) error {
				return errors.New("update error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockReservationRepo{
				findByIDFn:     tt.findByID,
				updateStatusFn: tt.updateFn,
			}
			svc := NewReservationService(repo, nil)
			err := svc.CancelReservation(ctx, 1)

			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Errorf("expected %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
