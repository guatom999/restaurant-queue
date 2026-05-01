package model

import "time"

type ReservationStatus string

const (
	ReservationStatusWaiting   ReservationStatus = "WAITING"
	ReservationStatusCalled    ReservationStatus = "CALLED"
	ReservationStatusSeated    ReservationStatus = "SEATED"
	ReservationStatusCompleted ReservationStatus = "COMPLETED"
	ReservationStatusCancelled ReservationStatus = "CANCELLED"
	ReservationStatusNoShow    ReservationStatus = "NO_SHOW"
)

type Reservation struct {
	ID               int64             `json:"id"`
	ReservationCode  string            `json:"reservation_code"`
	UserID           int64             `json:"user_id"`
	RestaurantID     int64             `json:"restaurant_id"`
	QueueNumber      int               `json:"queue_number"`
	PartySize        int               `json:"party_size"`
	Status           ReservationStatus `json:"status"`
	ReserveStartTime string            `json:"reserve_start_time"`
	ReservedForDate  time.Time         `json:"reserved_for_date"`
	ReservedAt       time.Time         `json:"reserved_at"`
	CalledAt         *time.Time        `json:"called_at"`
	CompletedAt      *time.Time        `json:"completed_at"`
	CancelledAt      *time.Time        `json:"cancelled_at"`
	Note             string            `json:"note"`
}
