package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/workshop/restaurant-api/internal/model"
)

// ─── mocks ───────────────────────────────────────────────────────────────────

type mockRestaurantRepo struct {
	findByIDFn              func(ctx context.Context, id int64) (*model.Restaurant, error)
	findAllFn               func(ctx context.Context) ([]*model.Restaurant, error)
	findBusinessHoursByDayFn func(ctx context.Context, restaurantID int64, dayOfWeek int) (*model.RestaurantBusinessHours, error)
}

func (m *mockRestaurantRepo) FindByID(ctx context.Context, id int64) (*model.Restaurant, error) {
	return m.findByIDFn(ctx, id)
}
func (m *mockRestaurantRepo) FindAll(ctx context.Context) ([]*model.Restaurant, error) {
	return m.findAllFn(ctx)
}
func (m *mockRestaurantRepo) FindBusinessHoursByDay(ctx context.Context, restaurantID int64, dayOfWeek int) (*model.RestaurantBusinessHours, error) {
	return m.findBusinessHoursByDayFn(ctx, restaurantID, dayOfWeek)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func sampleRestaurant() *model.Restaurant {
	return &model.Restaurant{ID: 1, Name: "Test Restaurant", IsActive: true}
}

func businessHours(open, close string, slotMin, capacity int) *model.RestaurantBusinessHours {
	parse := func(s string) time.Time {
		t, _ := time.Parse("15:04", s)
		return t
	}
	return &model.RestaurantBusinessHours{
		RestaurantID:        1,
		DayOfWeek:           1,
		OpenTime:            parse(open),
		CloseTime:           parse(close),
		SlotDurationMinutes: slotMin,
		MaxCapacityPerSlot:  capacity,
		IsActive:            true,
	}
}

// ─── ListRestaurants ──────────────────────────────────────────────────────────

func TestListRestaurants(t *testing.T) {
	ctx := context.Background()

	t.Run("returns all restaurants", func(t *testing.T) {
		want := []*model.Restaurant{sampleRestaurant(), {ID: 2, Name: "Another"}}
		repo := &mockRestaurantRepo{
			findAllFn: func(_ context.Context) ([]*model.Restaurant, error) { return want, nil },
		}
		svc := NewRestaurantService(repo, nil)
		got, err := svc.ListRestaurants(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Errorf("expected %d, got %d", len(want), len(got))
		}
	})

	t.Run("repo error propagates", func(t *testing.T) {
		repo := &mockRestaurantRepo{
			findAllFn: func(_ context.Context) ([]*model.Restaurant, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewRestaurantService(repo, nil)
		_, err := svc.ListRestaurants(ctx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

// ─── GetRestaurant ────────────────────────────────────────────────────────────

func TestGetRestaurant(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		findByID  func(context.Context, int64) (*model.Restaurant, error)
		wantErrIs error
		wantErr   bool
	}{
		{
			name: "success",
			findByID: func(_ context.Context, _ int64) (*model.Restaurant, error) {
				return sampleRestaurant(), nil
			},
		},
		{
			name: "not found returns sentinel",
			findByID: func(_ context.Context, _ int64) (*model.Restaurant, error) {
				return nil, nil
			},
			wantErrIs: ErrRestaurantNotFound,
		},
		{
			name: "repo error propagates",
			findByID: func(_ context.Context, _ int64) (*model.Restaurant, error) {
				return nil, errors.New("db error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRestaurantRepo{findByIDFn: tt.findByID}
			svc := NewRestaurantService(repo, nil)
			got, err := svc.GetRestaurant(ctx, 1)

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
				t.Error("expected restaurant, got nil")
			}
		})
	}
}

// ─── GetAvailability ──────────────────────────────────────────────────────────

func TestGetAvailability(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC) // Friday

	okRestaurantRepo := func() func(context.Context, int64) (*model.Restaurant, error) {
		return func(_ context.Context, _ int64) (*model.Restaurant, error) {
			return sampleRestaurant(), nil
		}
	}

	tests := []struct {
		name               string
		restaurantFn       func(context.Context, int64) (*model.Restaurant, error)
		bizHoursFn         func(context.Context, int64, int) (*model.RestaurantBusinessHours, error)
		reserveFn          func(context.Context, int64, time.Time) (int, error)
		findAvailabilityFn func(context.Context, int64, time.Time) ([]model.Reservation, error)
		wantErrIs          error
		wantErr            bool
		checkResult        func(*testing.T, *model.AvailabilityResult)
	}{
		{
			name: "restaurant not found",
			restaurantFn: func(_ context.Context, _ int64) (*model.Restaurant, error) {
				return nil, nil
			},
			wantErrIs: ErrRestaurantNotFound,
		},
		{
			name: "restaurant repo error",
			restaurantFn: func(_ context.Context, _ int64) (*model.Restaurant, error) {
				return nil, errors.New("db error")
			},
			wantErr: true,
		},
		{
			name:         "business hours repo error",
			restaurantFn: okRestaurantRepo(),
			bizHoursFn: func(_ context.Context, _ int64, _ int) (*model.RestaurantBusinessHours, error) {
				return nil, errors.New("bh error")
			},
			wantErr: true,
		},
		{
			name:         "closed day returns empty slots",
			restaurantFn: okRestaurantRepo(),
			bizHoursFn: func(_ context.Context, _ int64, _ int) (*model.RestaurantBusinessHours, error) {
				return nil, nil
			},
			checkResult: func(t *testing.T, r *model.AvailabilityResult) {
				if r.IsOpen {
					t.Error("expected closed")
				}
				if len(r.Slots) != 0 {
					t.Errorf("expected no slots, got %d", len(r.Slots))
				}
			},
		},
		{
			name:         "count active by date error",
			restaurantFn: okRestaurantRepo(),
			bizHoursFn: func(_ context.Context, _ int64, _ int) (*model.RestaurantBusinessHours, error) {
				return businessHours("09:00", "17:00", 60, 10), nil
			},
			reserveFn: func(_ context.Context, _ int64, _ time.Time) (int, error) {
				return 0, errors.New("count error")
			},
			wantErr: true,
		},
		{
			name:         "success with slots",
			restaurantFn: okRestaurantRepo(),
			bizHoursFn: func(_ context.Context, _ int64, _ int) (*model.RestaurantBusinessHours, error) {
				return businessHours("09:00", "11:00", 60, 5), nil
			},
			reserveFn: func(_ context.Context, _ int64, _ time.Time) (int, error) {
				return 3, nil
			},
			checkResult: func(t *testing.T, r *model.AvailabilityResult) {
				if !r.IsOpen {
					t.Error("expected open")
				}
				if len(r.Slots) != 2 {
					t.Errorf("expected 2 slots (09:00-10:00, 10:00-11:00), got %d", len(r.Slots))
				}
				if r.TotalCapacity != 10 { // 2 slots × 5 capacity
					t.Errorf("expected total capacity 10, got %d", r.TotalCapacity)
				}
				if r.TotalBooked != 3 {
					t.Errorf("expected total booked 3, got %d", r.TotalBooked)
				}
				if r.TotalAvailable != 7 {
					t.Errorf("expected available 7, got %d", r.TotalAvailable)
				}
			},
		},
		{
			name:         "booked exceeds capacity clamps to zero",
			restaurantFn: okRestaurantRepo(),
			bizHoursFn: func(_ context.Context, _ int64, _ int) (*model.RestaurantBusinessHours, error) {
				return businessHours("09:00", "10:00", 60, 2), nil
			},
			reserveFn: func(_ context.Context, _ int64, _ time.Time) (int, error) {
				return 99, nil
			},
			checkResult: func(t *testing.T, r *model.AvailabilityResult) {
				if r.TotalAvailable != 0 {
					t.Errorf("expected 0 available, got %d", r.TotalAvailable)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restaurantRepo := &mockRestaurantRepo{
				findByIDFn:               tt.restaurantFn,
				findBusinessHoursByDayFn: tt.bizHoursFn,
			}
			reserveRepo := &mockReservationRepo{
				countActiveByDateFn:      tt.reserveFn,
				findAvailabilityByDateFn: tt.findAvailabilityFn,
			}
			svc := NewRestaurantService(restaurantRepo, reserveRepo)
			result, err := svc.GetAvailability(ctx, 1, date)

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
				return
			}
			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

// ─── buildSlots ───────────────────────────────────────────────────────────────

func TestBuildSlots(t *testing.T) {
	parse := func(s string) time.Time {
		t, _ := time.Parse("15:04", s)
		return t
	}

	tests := []struct {
		name      string
		open      string
		close     string
		duration  int
		capacity  int
		wantCount int
		wantFirst model.TimeSlot
		wantLast  model.TimeSlot
	}{
		{
			name:      "two equal slots",
			open:      "09:00",
			close:     "11:00",
			duration:  60,
			capacity:  10,
			wantCount: 2,
			wantFirst: model.TimeSlot{StartTime: "09:00", EndTime: "10:00", Capacity: 10},
			wantLast:  model.TimeSlot{StartTime: "10:00", EndTime: "11:00", Capacity: 10},
		},
		{
			name:      "partial last slot clipped to close time",
			open:      "09:00",
			close:     "10:30",
			duration:  60,
			capacity:  5,
			wantCount: 2,
			wantFirst: model.TimeSlot{StartTime: "09:00", EndTime: "10:00", Capacity: 5},
			wantLast:  model.TimeSlot{StartTime: "10:00", EndTime: "10:30", Capacity: 5},
		},
		{
			name:      "empty when open equals close",
			open:      "09:00",
			close:     "09:00",
			duration:  60,
			capacity:  10,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slots := buildSlots(parse(tt.open), parse(tt.close), tt.duration, tt.capacity, make(map[string]int))
			if len(slots) != tt.wantCount {
				t.Fatalf("expected %d slots, got %d", tt.wantCount, len(slots))
			}
			if tt.wantCount == 0 {
				return
			}
			first := slots[0]
			if first != tt.wantFirst {
				t.Errorf("first slot: expected %+v, got %+v", tt.wantFirst, first)
			}
			last := slots[len(slots)-1]
			if last != tt.wantLast {
				t.Errorf("last slot: expected %+v, got %+v", tt.wantLast, last)
			}
		})
	}
}
