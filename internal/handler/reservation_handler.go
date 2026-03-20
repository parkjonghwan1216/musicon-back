package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"musicon-back/internal/service"
)

type ReservationHandler struct {
	reservationService *service.ReservationService
}

func NewReservationHandler(reservationService *service.ReservationService) *ReservationHandler {
	return &ReservationHandler{reservationService: reservationService}
}

type createReservationRequest struct {
	ExpoPushToken string `json:"expo_push_token"`
	Artist        string `json:"artist"`
	Title         string `json:"title"`
}

type updateReservationRequest struct {
	ExpoPushToken string `json:"expo_push_token"`
	Artist        string `json:"artist"`
	Title         string `json:"title"`
}

// Create godoc
// @Summary     예약 등록
// @Description 선곡 예약을 등록합니다
// @Tags        reservations
// @Accept      json
// @Produce     json
// @Param       body body     createReservationRequest true "예약 등록 요청"
// @Success     201  {object} map[string]interface{}
// @Failure     400  {object} map[string]interface{}
// @Failure     500  {object} map[string]interface{}
// @Router      /api/reservations [post]
func (h *ReservationHandler) Create(c *fiber.Ctx) error {
	var req createReservationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid request body",
		})
	}

	if req.ExpoPushToken == "" || req.Artist == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "expo_push_token and artist are required",
		})
	}

	reservation, err := h.reservationService.Create(c.Context(), req.ExpoPushToken, req.Artist, req.Title)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to create reservation",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    reservation,
	})
}

// List godoc
// @Summary     예약 목록 조회
// @Description 디바이스의 선곡 예약 목록을 조회합니다
// @Tags        reservations
// @Produce     json
// @Param       token query    string true "Expo Push Token"
// @Success     200   {object} map[string]interface{}
// @Failure     400   {object} map[string]interface{}
// @Failure     500   {object} map[string]interface{}
// @Router      /api/reservations [get]
func (h *ReservationHandler) List(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "token query parameter is required",
		})
	}

	reservations, err := h.reservationService.ListByDevice(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to list reservations",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    reservations,
		"meta": fiber.Map{
			"count": len(reservations),
		},
	})
}

// Update godoc
// @Summary     예약 수정
// @Description 선곡 예약을 수정합니다
// @Tags        reservations
// @Accept      json
// @Produce     json
// @Param       id   path     int                      true "예약 ID"
// @Param       body body     updateReservationRequest true "예약 수정 요청"
// @Success     200  {object} map[string]interface{}
// @Failure     400  {object} map[string]interface{}
// @Failure     500  {object} map[string]interface{}
// @Router      /api/reservations/{id} [put]
func (h *ReservationHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid reservation id",
		})
	}

	var req updateReservationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid request body",
		})
	}

	if req.ExpoPushToken == "" || req.Artist == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "expo_push_token and artist are required",
		})
	}

	reservation, err := h.reservationService.Update(c.Context(), req.ExpoPushToken, id, req.Artist, req.Title)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to update reservation",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    reservation,
	})
}

// Delete godoc
// @Summary     예약 삭제
// @Description 선곡 예약을 삭제합니다
// @Tags        reservations
// @Produce     json
// @Param       id    path     int    true "예약 ID"
// @Param       token query    string true "Expo Push Token"
// @Success     200   {object} map[string]interface{}
// @Failure     400   {object} map[string]interface{}
// @Failure     500   {object} map[string]interface{}
// @Router      /api/reservations/{id} [delete]
func (h *ReservationHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid reservation id",
		})
	}

	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "token query parameter is required",
		})
	}

	if err := h.reservationService.Delete(c.Context(), token, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to delete reservation",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}
