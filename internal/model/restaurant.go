package model

import "time"

type Restaurant struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Phone     string    `json:"phone"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type RestaurantBusinessHours struct {
	ID                  int64     `json:"id"`
	RestaurantID        int64     `json:"restaurant_id"`
	DayOfWeek           int       `json:"day_of_week"`
	OpenTime            time.Time `json:"open_time"`
	CloseTime           time.Time `json:"close_time"`
	SlotDurationMinutes int       `json:"slot_duration_minutes"`
	MaxCapacityPerSlot  int       `json:"max_capacity_per_slot"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
}
