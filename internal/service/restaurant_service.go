package service

import (
	"context"
	"errors"

	"github.com/workshop/restaurant-api/internal/model"
	"github.com/workshop/restaurant-api/internal/repository"
)

var ErrRestaurantNotFound = errors.New("restaurant not found")

type RestaurantService interface {
	ListRestaurants(ctx context.Context) ([]*model.Restaurant, error)
	GetRestaurant(ctx context.Context, id int64) (*model.Restaurant, error)
}

type restaurantService struct {
	repo repository.RestaurantRepository
}

func NewRestaurantService(repo repository.RestaurantRepository) RestaurantService {
	return &restaurantService{repo: repo}
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
