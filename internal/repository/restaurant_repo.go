package repository

import (
	"context"
	"database/sql"

	"github.com/workshop/restaurant-api/internal/model"
)

type RestaurantRepository interface {
	FindByID(ctx context.Context, id int64) (*model.Restaurant, error)
	FindAll(ctx context.Context) ([]*model.Restaurant, error)
}

type restaurantRepo struct {
	db *sql.DB
}

func NewRestaurantRepo(db *sql.DB) RestaurantRepository {
	return &restaurantRepo{db: db}
}

func (r *restaurantRepo) FindByID(ctx context.Context, id int64) (*model.Restaurant, error) {
	var res model.Restaurant
	err := r.db.QueryRowContext(ctx, `
		SELECT id, code, name, address, phone, is_active, created_at
		FROM restaurants
		WHERE id = $1
	`, id).Scan(&res.ID, &res.Code, &res.Name, &res.Address, &res.Phone, &res.IsActive, &res.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *restaurantRepo) FindAll(ctx context.Context) ([]*model.Restaurant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, name, address, phone, is_active, created_at
		FROM restaurants
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*model.Restaurant
	for rows.Next() {
		var restaurant model.Restaurant
		if err := rows.Scan(&restaurant.ID, &restaurant.Code, &restaurant.Name, &restaurant.Address, &restaurant.Phone, &restaurant.IsActive, &restaurant.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, &restaurant)
	}
	return res, rows.Err()
}
