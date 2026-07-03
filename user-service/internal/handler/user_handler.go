package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/user-service/internal/domain"
)

// UserService defines the service contract used by the user handler.
type UserService interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, page, pageSize int) ([]*domain.User, int, error)
}

// Handler handles HTTP requests for users.
type Handler struct {
	svc UserService
}

// New creates a new user handler.
func New(svc UserService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers user routes on the Echo instance.
func RegisterRoutes(e *echo.Echo, h *Handler) {
	v1 := e.Group("/v1")
	v1.GET("/users/:id", h.GetByID)
	v1.POST("/users", h.Create)
	v1.PUT("/users/:id", h.Update)
	v1.DELETE("/users/:id", h.Delete)
	v1.GET("/users", h.List)

	// Internal endpoints for service-to-service calls
	internal := e.Group("/v1/internal")
	internal.GET("/users/:id", h.GetByID)
}

func (h *Handler) GetByID(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing user id"})
	}

	user, err := h.svc.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, user.ToCreateResponse())
}

func (h *Handler) Create(c echo.Context) error {
	var req domain.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
	}

	// Basic validation
	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email is required"})
	}
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	now := time.Now().UTC()
	user := req.ToUser(uuid.New().String(), now)

	if err := h.svc.Create(c.Request().Context(), user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, user.ToCreateResponse())
}

func (h *Handler) Update(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing user id"})
	}

	var req domain.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
	}

	user, err := h.svc.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	user.Name = req.Name
	user.Bio = req.Bio
	user.PhotoURLs = req.PhotoURLs
	user.UpdatedAt = time.Now().UTC()

	if err := h.svc.Update(c.Request().Context(), user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, user.ToCreateResponse())
}

func (h *Handler) Delete(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing user id"})
	}

	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) List(c echo.Context) error {
	users, total, err := h.svc.List(c.Request().Context(), 1, 20)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	responses := make([]*domain.CreateUserResponse, 0, len(users))
	for _, u := range users {
		responses = append(responses, u.ToCreateResponse())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": responses,
		"total": total,
	})
}
