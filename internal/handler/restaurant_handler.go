package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/workshop/restaurant-api/internal/service"
)

type RestaurantHandler struct {
	svc service.RestaurantService
}

func NewRestaurantHandler(svc service.RestaurantService) *RestaurantHandler {
	return &RestaurantHandler{svc: svc}
}

func (h *RestaurantHandler) RegisterRoutes(app *fiber.App) {
	app.Get("/api/v1/restaurants", h.list)
	app.Get("/api/v1/restaurants/:id", h.getByID)
	app.Get("/api/v1/restaurants/:id/availability", h.getAvailability)
}

func (h *RestaurantHandler) list(c *fiber.Ctx) error {
	restaurants, err := h.svc.ListRestaurants(c.Context())
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, err.Error())
	}
	return writeJSON(c, fiber.StatusOK, restaurants)
}

func (h *RestaurantHandler) getByID(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid id")
	}
	restaurant, err := h.svc.GetRestaurant(c.Context(), id)
	if err == service.ErrRestaurantNotFound {
		return writeError(c, fiber.StatusNotFound, err.Error())
	}
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, err.Error())
	}
	return writeJSON(c, fiber.StatusOK, restaurant)
}

func (h *RestaurantHandler) getAvailability(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid id")
	}

	dateStr := c.Query("date")
	if dateStr == "" {
		return writeError(c, fiber.StatusBadRequest, "date query param is required (YYYY-MM-DD)")
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid date format, use YYYY-MM-DD")
	}

	result, err := h.svc.GetAvailability(c.Context(), id, date)
	if err == service.ErrRestaurantNotFound {
		return writeError(c, fiber.StatusNotFound, err.Error())
	}
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, err.Error())
	}
	return writeJSON(c, fiber.StatusOK, result)
}
