package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/workshop/restaurant-api/internal/model"
	"github.com/workshop/restaurant-api/internal/service"
)

type ReservationHandler struct {
	svc service.ReservationService
}

func NewReservationHandler(svc service.ReservationService) *ReservationHandler {
	return &ReservationHandler{svc: svc}
}

func (h *ReservationHandler) RegisterRoutes(app *fiber.App) {
	app.Get("/api/v1/restaurants/:restaurantID/reservations", h.list)
	app.Post("/api/v1/restaurants/:restaurantID/reservations", h.book)
	app.Get("/api/v1/reservations/:id", h.getByID)
	app.Patch("/api/v1/reservations/:id/confirm", h.confirm)
	app.Patch("/api/v1/reservations/:id/cancel", h.cancel)
}

type bookRequest struct {
	UserID          int64  `json:"user_id"`
	PartySize       int    `json:"party_size"`
	ReservedForDate string `json:"reserved_for_date"` // "2006-01-02"
	Note            string `json:"note"`
}

func (h *ReservationHandler) list(c *fiber.Ctx) error {
	restaurantID, err := strconv.ParseInt(c.Params("restaurantID"), 10, 64)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid restaurant id")
	}
	reservations, err := h.svc.ListReservations(c.Context(), restaurantID)
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, err.Error())
	}
	return writeJSON(c, fiber.StatusOK, reservations)
}

func (h *ReservationHandler) book(c *fiber.Ctx) error {
	restaurantID, err := strconv.ParseInt(c.Params("restaurantID"), 10, 64)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid restaurant id")
	}

	var req bookRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid request body")
	}

	reservedFor, err := time.Parse("2006-01-02", req.ReservedForDate)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "reserved_for_date must be in YYYY-MM-DD format")
	}

	res := model.Reservation{
		UserID:          req.UserID,
		RestaurantID:    restaurantID,
		PartySize:       req.PartySize,
		ReservedForDate: reservedFor,
		Note:            req.Note,
	}

	id, err := h.svc.BookReservation(c.Context(), &res)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id})
}

func (h *ReservationHandler) getByID(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid id")
	}
	res, err := h.svc.GetReservation(c.Context(), id)
	if errors.Is(err, service.ErrReservationNotFound) {
		return writeError(c, fiber.StatusNotFound, err.Error())
	}
	if err != nil {
		return writeError(c, fiber.StatusInternalServerError, err.Error())
	}
	return writeJSON(c, fiber.StatusOK, res)
}

func (h *ReservationHandler) confirm(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid id")
	}
	if err := h.svc.ConfirmReservation(c.Context(), id); errors.Is(err, service.ErrReservationNotFound) {
		return writeError(c, fiber.StatusNotFound, err.Error())
	} else if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ReservationHandler) cancel(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return writeError(c, fiber.StatusBadRequest, "invalid id")
	}
	if err := h.svc.CancelReservation(c.Context(), id); errors.Is(err, service.ErrReservationNotFound) {
		return writeError(c, fiber.StatusNotFound, err.Error())
	} else if err != nil {
		return writeError(c, fiber.StatusBadRequest, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
