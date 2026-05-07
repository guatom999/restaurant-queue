package service

import (
	"context"
	"errors"
	"log"
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
		log.Printf("Error CountActiveByDate for restaurant %d on %s: %v", restaurantID, date.Format("2006-01-02"), err)

		return nil, err
	}

	reserved, err := s.reserveRepo.FindAvailabilityByDate(ctx, restaurantID, date)
	if err != nil {
		log.Printf("Error finding availability for restaurant %d on %s: %v", restaurantID, date.Format("2006-01-02"), err)
		return nil, err
	}

	reservedTimeSlot := buildSlotsOfRestaurantByTime(reserved)

	slots := buildSlots(bh.OpenTime, bh.CloseTime, bh.SlotDurationMinutes, bh.MaxCapacityPerSlot, reservedTimeSlot)
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

func buildSlotsOfRestaurantByTime(reserved []model.Reservation) map[string]int {
	reservedTimeSlot := make(map[string]int, 0)

	for _, i := range reserved {
		if _, ok := reservedTimeSlot[i.ReserveStartTime]; ok {
			reservedTimeSlot[i.ReserveStartTime]++
		} else {
			reservedTimeSlot[i.ReserveStartTime] = 1
		}
	}

	return reservedTimeSlot
}

func buildSlots(openTime, closeTime time.Time, durationMinutes, capacityPerSlot int, reservedTimeSlot map[string]int) []model.TimeSlot {
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
			Capacity: func(startTime string) int {

				if reserved, ok := reservedTimeSlot[startTime]; ok {
					return capacityPerSlot - reserved
				}

				return capacityPerSlot
			}(cur.Format("15:04")),
		})
		cur = end
	}
	return slots
}
