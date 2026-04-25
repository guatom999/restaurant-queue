package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/workshop/restaurant-api/internal/handler"
)

func (s *Server) registerRoutes() {
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	handler.NewReservationHandler(s.reservationSvc).RegisterRoutes(s.app)
	handler.NewRestaurantHandler(s.restaurantSvc).RegisterRoutes(s.app)
}
