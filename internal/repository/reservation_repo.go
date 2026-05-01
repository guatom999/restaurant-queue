package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/workshop/restaurant-api/internal/model"
)

type ReservationRepository interface {
	FindByID(ctx context.Context, id int64) (*model.Reservation, error)
	FindByRestaurant(ctx context.Context, restaurantID int64) ([]model.Reservation, error)
	Create(ctx context.Context, r *model.Reservation) (int64, error)
	UpdateStatus(ctx context.Context, id int64, status model.ReservationStatus) error
	CountActiveByDate(ctx context.Context, restaurantID int64, date time.Time) (int, error)
}

type reservationRepo struct {
	db *sql.DB
}

func NewReservationRepo(db *sql.DB) ReservationRepository {
	return &reservationRepo{db: db}
}

func (r *reservationRepo) FindByID(ctx context.Context, id int64) (*model.Reservation, error) {
	var res model.Reservation
	err := r.db.QueryRowContext(ctx,
		`SELECT id, reservation_code, user_id, restaurant_id, queue_number,
		        party_size, status, reserve_start_time, reserved_for_date, reserved_at,
		        called_at, completed_at, cancelled_at, note
		 FROM reservations WHERE id = $1`, id).
		Scan(&res.ID, &res.ReservationCode, &res.UserID, &res.RestaurantID, &res.QueueNumber,
			&res.PartySize, &res.Status, &res.ReserveStartTime, &res.ReservedForDate, &res.ReservedAt,
			&res.CalledAt, &res.CompletedAt, &res.CancelledAt, &res.Note)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *reservationRepo) FindByRestaurant(ctx context.Context, restaurantID int64) ([]model.Reservation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, reservation_code, user_id, restaurant_id, queue_number,
		        party_size, status, reserve_start_time, reserved_for_date, reserved_at,
		        called_at, completed_at, cancelled_at, note
		 FROM reservations WHERE restaurant_id = $1
		 ORDER BY reserved_for_date, queue_number`, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []model.Reservation
	for rows.Next() {
		var res model.Reservation
		if err := rows.Scan(&res.ID, &res.ReservationCode, &res.UserID, &res.RestaurantID, &res.QueueNumber,
			&res.PartySize, &res.Status, &res.ReserveStartTime, &res.ReservedForDate, &res.ReservedAt,
			&res.CalledAt, &res.CompletedAt, &res.CancelledAt, &res.Note); err != nil {
			return nil, err
		}
		reservations = append(reservations, res)
	}
	return reservations, rows.Err()
}

func (r *reservationRepo) Create(ctx context.Context, res *model.Reservation) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// advisory lock per restaurant to serialize concurrent bookings on the same day
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, res.RestaurantID); err != nil {
		return 0, err
	}

	var queueNum int
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(queue_number), 0) + 1
		 FROM reservations
		 WHERE restaurant_id = $1 AND reserved_for_date = $2`,
		res.RestaurantID, res.ReservedForDate.Format("2006-01-02")).Scan(&queueNum); err != nil {
		return 0, err
	}

	res.QueueNumber = queueNum
	res.ReservationCode = fmt.Sprintf("Q%04d", queueNum)

	var id int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO reservations
		        (reservation_code, user_id, restaurant_id, queue_number,
		         party_size, status, reserve_start_time, reserved_for_date, note)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		res.ReservationCode, res.UserID, res.RestaurantID, res.QueueNumber,
		res.PartySize, res.Status, res.ReserveStartTime, res.ReservedForDate, res.Note).Scan(&id); err != nil {
		return 0, err
	}

	return id, tx.Commit()
}

func (r *reservationRepo) UpdateStatus(ctx context.Context, id int64, status model.ReservationStatus) error {
	var query string
	switch status {
	case model.ReservationStatusCalled:
		query = `UPDATE reservations SET status=$1, called_at=NOW() WHERE id=$2`
	case model.ReservationStatusCompleted:
		query = `UPDATE reservations SET status=$1, completed_at=NOW() WHERE id=$2`
	case model.ReservationStatusCancelled:
		query = `UPDATE reservations SET status=$1, cancelled_at=NOW() WHERE id=$2`
	default:
		query = `UPDATE reservations SET status=$1 WHERE id=$2`
	}
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *reservationRepo) CountActiveByDate(ctx context.Context, restaurantID int64, date time.Time) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM reservations
		 WHERE restaurant_id = $1
		   AND reserved_for_date = $2
		   AND status IN ('WAITING', 'CALLED', 'SEATED')`,
		restaurantID, date.Format("2006-01-02")).Scan(&count)
	return count, err
}
