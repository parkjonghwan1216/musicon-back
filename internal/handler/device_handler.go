package handler

import (
	"github.com/gofiber/fiber/v2"

	"musicon-back/internal/service"
)

type DeviceHandler struct {
	deviceService *service.DeviceService
}

func NewDeviceHandler(deviceService *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceService: deviceService}
}

type registerDeviceRequest struct {
	ExpoPushToken string `json:"expo_push_token"`
	Platform      string `json:"platform"`
}

// Register godoc
// @Summary     디바이스 등록
// @Description Expo Push Token으로 디바이스를 등록합니다
// @Tags        devices
// @Accept      json
// @Produce     json
// @Param       body body     registerDeviceRequest true "디바이스 등록 요청"
// @Success     200  {object} map[string]interface{}
// @Failure     400  {object} map[string]interface{}
// @Failure     500  {object} map[string]interface{}
// @Router      /api/devices/register [post]
func (h *DeviceHandler) Register(c *fiber.Ctx) error {
	var req registerDeviceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid request body",
		})
	}

	if req.ExpoPushToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "expo_push_token is required",
		})
	}

	device, err := h.deviceService.Register(c.Context(), req.ExpoPushToken, req.Platform)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to register device",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    device,
	})
}
