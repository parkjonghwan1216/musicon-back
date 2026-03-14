package handler

import (
	"log"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"musicon-back/internal/service"
)

type MusicHandler struct {
	musicAuthSvc  *service.MusicAuthService
	musicSyncSvc  *service.MusicSyncService
	musicQuerySvc *service.MusicQueryService
	baseURL       string
}

func NewMusicHandler(
	musicAuthSvc *service.MusicAuthService,
	musicSyncSvc *service.MusicSyncService,
	musicQuerySvc *service.MusicQueryService,
	baseURL string,
) *MusicHandler {
	return &MusicHandler{
		musicAuthSvc:  musicAuthSvc,
		musicSyncSvc:  musicSyncSvc,
		musicQuerySvc: musicQuerySvc,
		baseURL:       baseURL,
	}
}

type connectRequest struct {
	Code          string `json:"code"`
	RedirectURI   string `json:"redirect_uri"`
	ExpoPushToken string `json:"expo_push_token"`
}

// ConnectSpotify godoc
// @Summary     Spotify 계정 연결
// @Description Spotify OAuth 코드로 계정을 연결합니다
// @Tags        music
// @Accept      json
// @Produce     json
// @Param       body body     connectRequest true "Spotify 연결 요청"
// @Success     200  {object} map[string]interface{}
// @Failure     400  {object} map[string]interface{}
// @Failure     500  {object} map[string]interface{}
// @Router      /api/music/spotify/connect [post]
func (h *MusicHandler) ConnectSpotify(c *fiber.Ctx) error {
	return h.connect(c, "spotify")
}

// ConnectYouTube godoc
// @Summary     YouTube 계정 연결
// @Description YouTube/Google OAuth 코드로 계정을 연결합니다
// @Tags        music
// @Accept      json
// @Produce     json
// @Param       body body     connectRequest true "YouTube 연결 요청"
// @Success     200  {object} map[string]interface{}
// @Failure     400  {object} map[string]interface{}
// @Failure     500  {object} map[string]interface{}
// @Router      /api/music/youtube/connect [post]
func (h *MusicHandler) ConnectYouTube(c *fiber.Ctx) error {
	return h.connect(c, "youtube")
}

func (h *MusicHandler) connect(c *fiber.Ctx, providerName string) error {
	var req connectRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid request body",
		})
	}

	if req.Code == "" || req.RedirectURI == "" || req.ExpoPushToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "code, redirect_uri, and expo_push_token are required",
		})
	}

	account, err := h.musicAuthSvc.Connect(c.Context(), providerName, req.Code, req.RedirectURI, req.ExpoPushToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to connect " + providerName,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    account,
	})
}

// YouTubeCallback godoc
// @Summary     YouTube OAuth 콜백
// @Description Google OAuth 인증 후 리디렉트되는 콜백 엔드포인트. 토큰 교환 후 앱으로 리디렉트합니다.
// @Tags        auth
// @Param       code  query string false "Authorization code"
// @Param       state query string false "Expo Push Token (URL-encoded)"
// @Param       error query string false "OAuth error"
// @Success     302
// @Router      /api/auth/youtube/callback [get]
func (h *MusicHandler) YouTubeCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")
	oauthError := c.Query("error")

	if oauthError != "" {
		log.Printf("YouTube OAuth error: %s", oauthError)
		return c.Redirect("musicon://auth-error?provider=youtube&error=" + url.QueryEscape(oauthError))
	}

	if code == "" || state == "" {
		return c.Redirect("musicon://auth-error?provider=youtube&error=" + url.QueryEscape("missing code or state"))
	}

	expoPushToken := state
	redirectURI := h.baseURL + "/api/auth/youtube/callback"

	account, err := h.musicAuthSvc.Connect(c.Context(), "youtube", code, redirectURI, expoPushToken)
	if err != nil {
		log.Printf("YouTube connect failed: %v", err)
		return c.Redirect("musicon://auth-error?provider=youtube&error=" + url.QueryEscape("연결에 실패했습니다"))
	}

	return c.Redirect("musicon://auth-success?provider=youtube&name=" + url.QueryEscape(account.DisplayName))
}

// ListAccounts godoc
// @Summary     연결된 음악 계정 목록
// @Description 디바이스에 연결된 음악 서비스 계정 목록을 조회합니다
// @Tags        music
// @Produce     json
// @Param       token query    string true "Expo Push Token"
// @Success     200   {object} map[string]interface{}
// @Failure     400   {object} map[string]interface{}
// @Failure     500   {object} map[string]interface{}
// @Router      /api/music/accounts [get]
func (h *MusicHandler) ListAccounts(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "token query parameter is required",
		})
	}

	accounts, err := h.musicAuthSvc.ListAccounts(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to list accounts",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    accounts,
		"meta": fiber.Map{
			"count": len(accounts),
		},
	})
}

// DisconnectAccount godoc
// @Summary     음악 계정 연결 해제
// @Description 특정 음악 서비스 계정 연결을 해제합니다
// @Tags        music
// @Produce     json
// @Param       provider path     string true "프로바이더 (spotify/youtube)"
// @Param       token    query    string true "Expo Push Token"
// @Success     200      {object} map[string]interface{}
// @Failure     400      {object} map[string]interface{}
// @Failure     500      {object} map[string]interface{}
// @Router      /api/music/accounts/{provider} [delete]
func (h *MusicHandler) DisconnectAccount(c *fiber.Ctx) error {
	providerName := c.Params("provider")
	if providerName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "provider parameter is required",
		})
	}

	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "token query parameter is required",
		})
	}

	if err := h.musicAuthSvc.Disconnect(c.Context(), token, providerName); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to disconnect " + providerName,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// SyncTracks godoc
// @Summary     트랙 동기화
// @Description 연결된 음악 서비스에서 트랙을 가져와 TJ DB와 매칭합니다
// @Tags        music
// @Produce     json
// @Param       token query    string true "Expo Push Token"
// @Success     200   {object} map[string]interface{}
// @Failure     400   {object} map[string]interface{}
// @Failure     500   {object} map[string]interface{}
// @Router      /api/music/sync [post]
func (h *MusicHandler) SyncTracks(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "token query parameter is required",
		})
	}

	results, err := h.musicSyncSvc.Sync(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to sync tracks",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    results,
	})
}

// GetMatches godoc
// @Summary     매칭 결과 조회
// @Description TJ DB와 매칭된 트랙 결과를 조회합니다
// @Tags        music
// @Produce     json
// @Param       token  query    string true  "Expo Push Token"
// @Param       limit  query    int    false "결과 수 (기본 20, 최대 100)"
// @Param       offset query    int    false "오프셋 (기본 0)"
// @Success     200    {object} map[string]interface{}
// @Failure     400    {object} map[string]interface{}
// @Failure     500    {object} map[string]interface{}
// @Router      /api/music/matches [get]
func (h *MusicHandler) GetMatches(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "token query parameter is required",
		})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	matches, err := h.musicQuerySvc.GetMatches(c.Context(), token, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to get matches",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    matches,
		"meta": fiber.Map{
			"limit":  limit,
			"offset": offset,
			"count":  len(matches),
		},
	})
}
