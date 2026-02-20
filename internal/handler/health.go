package handler

import (
	"github.com/gofiber/fiber/v2"
)

// HealthCheck godoc
// @Summary     헬스체크
// @Description 서버 상태 확인
// @Tags        system
// @Produce     json
// @Success     200 {object} map[string]string
// @Router      /health [get]
func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
	})
}
