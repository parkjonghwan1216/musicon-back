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
