package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/workshop/restaurant-api/internal/service"
)

type Server struct {
	app            *fiber.App
	reservationSvc service.ReservationService
	restaurantSvc  service.RestaurantService
}

func New(reservationSvc service.ReservationService, restaurantSvc service.RestaurantService) *Server {
	s := &Server{
		app:            fiber.New(),
		reservationSvc: reservationSvc,
		restaurantSvc:  restaurantSvc,
	}
	s.registerRoutes()
	return s
}

func (s *Server) Start(addr string) error {
	return s.app.Listen(addr)
}
