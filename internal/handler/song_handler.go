package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"musicon-back/internal/service"
)

type SongHandler struct {
	songService *service.SongService
}

func NewSongHandler(songService *service.SongService) *SongHandler {
	return &SongHandler{songService: songService}
}

// Search godoc
// @Summary     곡 검색
// @Description 제목, 가수, 초성으로 곡을 검색합니다
// @Tags        songs
// @Produce     json
// @Param       q      query    string true  "검색어 (제목/가수/초성)"
// @Param       limit  query    int    false "결과 수 (기본 20, 최대 100)"
// @Param       offset query    int    false "오프셋 (기본 0)"
// @Success     200    {object} map[string]interface{}
// @Failure     400    {object} map[string]interface{}
// @Failure     500    {object} map[string]interface{}
// @Router      /api/songs/search [get]
func (h *SongHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "query parameter 'q' is required",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	songs, err := h.songService.Search(c.Context(), query, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to search songs",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    songs,
		"meta": fiber.Map{
			"query":  query,
			"limit":  limit,
			"offset": offset,
			"count":  len(songs),
		},
	})
}

// FindByTjNumber godoc
// @Summary     TJ 번호로 곡 조회
// @Description TJ 노래방 번호로 곡을 조회합니다
// @Tags        songs
// @Produce     json
// @Param       number path     int true "TJ 노래방 번호"
// @Success     200    {object} map[string]interface{}
// @Failure     400    {object} map[string]interface{}
// @Failure     404    {object} map[string]interface{}
// @Failure     500    {object} map[string]interface{}
// @Router      /api/songs/{number} [get]
func (h *SongHandler) FindByTjNumber(c *fiber.Ctx) error {
	tjNumber, err := strconv.Atoi(c.Params("number"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid TJ number",
		})
	}

	song, err := h.songService.FindByTjNumber(c.Context(), tjNumber)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to find song",
		})
	}

	if song == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "song not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    song,
	})
}
