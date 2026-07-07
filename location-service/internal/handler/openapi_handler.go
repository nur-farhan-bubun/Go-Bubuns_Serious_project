package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/location-service/api"
	"github.com/ride-sharing/location-service/internal/config"
	"github.com/ride-sharing/location-service/internal/domain"
	"github.com/ride-sharing/location-service/internal/service"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// errResponse is a local error response type used when the generated ErrorResponse
// is not available (not referenced by any operation in the spec).
type errResponse struct {
	Error string `json:"error"`
}

// Compile-time check that OpenAPIHandler implements api.ServerInterface.
var _ api.ServerInterface = (*OpenAPIHandler)(nil)

// OpenAPIHandler implements the generated api.ServerInterface.
type OpenAPIHandler struct {
	cfg     *config.Config
	svc     *service.Service
}

// NewOpenAPIHandler creates a new handler satisfying the generated ServerInterface.
func NewOpenAPIHandler(cfg *config.Config, svc *service.Service) *OpenAPIHandler {
	return &OpenAPIHandler{cfg: cfg, svc: svc}
}

// ─── Location ─────────────────────────────────────────────────────────

// UpdateLocation handles PUT /v1/location.
func (h *OpenAPIHandler) UpdateLocation(ctx echo.Context) error {
	var req api.UpdateLocationRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errResponse{Error: "invalid request"})
	}

	loc := &domain.Location{
		UserID:    req.UserId,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}

	if err := h.svc.UpdateLocation(ctx.Request().Context(), loc); err != nil {
		return ctx.JSON(http.StatusInternalServerError, errResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, nil)
}

// GetLocation handles GET /v1/location.
func (h *OpenAPIHandler) GetLocation(ctx echo.Context) error {
	// For now, user_id is passed as a query param or header
	userID := ctx.QueryParam("user_id")
	if userID == "" {
		userID = ctx.Request().Header.Get("X-User-ID")
	}
	if userID == "" {
		return ctx.JSON(http.StatusBadRequest, errResponse{Error: "user_id is required"})
	}

	loc, err := h.svc.GetLocation(ctx.Request().Context(), userID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, api.LocationResponse{
		UserId:    loc.UserID,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		UpdatedAt: loc.UpdatedAt,
	})
}

// ─── Map Posts ────────────────────────────────────────────────────────

// CreatePost handles POST /v1/posts.
func (h *OpenAPIHandler) CreatePost(ctx echo.Context) error {
	var req api.CreatePostRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errResponse{Error: "invalid request"})
	}

	post := &domain.MapPost{
		UserID:    req.UserId,
		Title:     req.Title,
		Content:   getStringPtr(req.Content),
		Category:  string(req.Category),
		ImageURLs: getStringSlice(req.ImageUrls),
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}

	created, err := h.svc.CreatePost(ctx.Request().Context(), post)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusCreated, toPostResponse(created))
}

// ListPosts handles GET /v1/posts.
func (h *OpenAPIHandler) ListPosts(ctx echo.Context, params api.ListPostsParams) error {
	lat := float64(0)
	lng := float64(0)
	radius := float64(0)
	category := ""

	if params.Lat != nil {
		lat = *params.Lat
	}
	if params.Lng != nil {
		lng = *params.Lng
	}
	if params.Radius != nil {
		radius = *params.Radius
	}
	if params.Category != nil {
		category = string(*params.Category)
	}

	posts, err := h.svc.ListPosts(ctx.Request().Context(), lat, lng, radius, category)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errResponse{Error: err.Error()})
	}

	responses := make([]api.PostResponse, 0, len(posts))
	for _, p := range posts {
		responses = append(responses, toPostResponse(p))
	}

	return ctx.JSON(http.StatusOK, responses)
}

// GetPost handles GET /v1/posts/{postId}.
func (h *OpenAPIHandler) GetPost(ctx echo.Context, postId openapi_types.UUID) error {
	post, err := h.svc.GetPost(ctx.Request().Context(), postId.String())
	if err != nil {
		return ctx.JSON(http.StatusNotFound, errResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, toPostResponse(post))
}

// UpdatePost handles PUT /v1/posts/{postId}.
func (h *OpenAPIHandler) UpdatePost(ctx echo.Context, postId openapi_types.UUID) error {
	var req api.UpdatePostRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errResponse{Error: "invalid request"})
	}

	post := &domain.MapPost{
		ID:       postId.String(),
		Title:    getStringPtr(req.Title),
		Content:  getStringPtr(req.Content),
		Category: getCategoryStr(req.Category),
	}

	if req.ImageUrls != nil {
		post.ImageURLs = *req.ImageUrls
	}

	updated, err := h.svc.UpdatePost(ctx.Request().Context(), post)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, toPostResponse(updated))
}

// DeletePost handles DELETE /v1/posts/{postId}.
func (h *OpenAPIHandler) DeletePost(ctx echo.Context, postId openapi_types.UUID) error {
	// Get user ID from header
	userID := ctx.Request().Header.Get("X-User-ID")
	if userID == "" {
		userID = ctx.QueryParam("user_id")
	}

	if err := h.svc.DeletePost(ctx.Request().Context(), postId.String(), userID); err != nil {
		return ctx.JSON(http.StatusInternalServerError, errResponse{Error: err.Error()})
	}

	return ctx.NoContent(http.StatusNoContent)
}

// ─── Helpers ──────────────────────────────────────────────────────────

func toPostResponse(post *domain.MapPost) api.PostResponse {
	return api.PostResponse{
		Id:        uuid.MustParse(post.ID),
		UserId:    post.UserID,
		Title:     post.Title,
		Content:   strPtr(post.Content),
		Category:  post.Category,
		ImageUrls: strSlicePtr(post.ImageURLs),
		Latitude:  post.Latitude,
		Longitude: post.Longitude,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
	}
}

func getStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getStringSlice(s *[]string) []string {
	if s == nil {
		return nil
	}
	return *s
}

func getCategoryStr(c *api.UpdatePostRequestCategory) string {
	if c == nil {
		return ""
	}
	return string(*c)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strSlicePtr(s []string) *[]string {
	if len(s) == 0 {
		return nil
	}
	return &s
}
