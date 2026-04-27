package service

import (
	"context"
	"errors"
	"time"

	"github.com/workshop/restaurant-api/internal/model"
	"github.com/workshop/restaurant-api/internal/repository"
)

var ErrRestaurantNotFound = errors.New("restaurant not found")

type RestaurantService interface {
	ListRestaurants(ctx context.Context) ([]*model.Restaurant, error)
	GetRestaurant(ctx context.Context, id int64) (*model.Restaurant, error)
	GetAvailability(ctx context.Context, restaurantID int64, date time.Time) (*model.AvailabilityResult, error)
}

type restaurantService struct {
	repo        repository.RestaurantRepository
	reserveRepo repository.ReservationRepository
}

func NewRestaurantService(repo repository.RestaurantRepository, reserveRepo repository.ReservationRepository) RestaurantService {
	return &restaurantService{repo: repo, reserveRepo: reserveRepo}
}

func (s *restaurantService) ListRestaurants(ctx context.Context) ([]*model.Restaurant, error) {
	return s.repo.FindAll(ctx)
}

func (s *restaurantService) GetRestaurant(ctx context.Context, id int64) (*model.Restaurant, error) {
	res, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrRestaurantNotFound
	}
	return res, nil
}

func (s *restaurantService) GetAvailability(ctx context.Context, restaurantID int64, date time.Time) (*model.AvailabilityResult, error) {
	restaurant, err := s.repo.FindByID(ctx, restaurantID)
	if err != nil {
		return nil, err
	}
	if restaurant == nil {
		return nil, ErrRestaurantNotFound
	}

	result := &model.AvailabilityResult{
		RestaurantID: restaurantID,
		Date:         date.Format("2006-01-02"),
		Slots:        []model.TimeSlot{},
	}

	bh, err := s.repo.FindBusinessHoursByDay(ctx, restaurantID, int(date.Weekday()))
	if err != nil {
		return nil, err
	}
	if bh == nil {
		return result, nil
	}

	booked, err := s.reserveRepo.CountActiveByDate(ctx, restaurantID, date)
	if err != nil {
		return nil, err
	}

	slots := buildSlots(bh.OpenTime, bh.CloseTime, bh.SlotDurationMinutes, bh.MaxCapacityPerSlot)
	totalCapacity := len(slots) * bh.MaxCapacityPerSlot
	available := totalCapacity - booked
	if available < 0 {
		available = 0
	}

	result.IsOpen = true
	result.Slots = slots
	result.TotalCapacity = totalCapacity
	result.TotalBooked = booked
	result.TotalAvailable = available
	return result, nil
}

func buildSlots(openTime, closeTime time.Time, durationMinutes, capacityPerSlot int) []model.TimeSlot {
	var slots []model.TimeSlot
	step := time.Duration(durationMinutes) * time.Minute
	cur := openTime
	for cur.Before(closeTime) {
		end := cur.Add(step)
		if end.After(closeTime) {
			end = closeTime
		}
		slots = append(slots, model.TimeSlot{
			StartTime: cur.Format("15:04"),
			EndTime:   end.Format("15:04"),
			Capacity:  capacityPerSlot,
		})
		cur = end
	}
	return slots
}
