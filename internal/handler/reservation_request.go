package handler

type reservationRequest struct {
	UserID           int64  `json:"user_id"`
	PartySize        int    `json:"party_size"`
	ReserveStartTime string `json:"reserve_start_time"` // "11:30"
	ReservedForDate  string `json:"reserved_for_date"`  // "2006-01-02"
	Note             string `json:"note"`
}
