package handler

import "github.com/gofiber/fiber/v2"

func writeJSON(c *fiber.Ctx, status int, v any) error {
	return c.Status(status).JSON(v)
}

func writeError(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": msg})
}
