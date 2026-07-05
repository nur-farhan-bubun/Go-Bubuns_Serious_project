package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ride-sharing/user-service/api"
	"github.com/ride-sharing/user-service/internal/domain"
)

// Compile-time check that OpenAPIHandler implements api.ServerInterface.
var _ api.ServerInterface = (*OpenAPIHandler)(nil)

// OpenAPIHandler implements the generated api.ServerInterface by delegating
// to the existing service layer. The business logic is unchanged from the
// original handlers (user_handler.go and auth_handler.go).
type OpenAPIHandler struct {
	userSvc UserService
	authSvc AuthService
}

// NewOpenAPIHandler creates a new handler that satisfies the generated
// ServerInterface.
func NewOpenAPIHandler(userSvc UserService, authSvc AuthService) *OpenAPIHandler {
	return &OpenAPIHandler{
		userSvc: userSvc,
		authSvc: authSvc,
	}
}

// ─── User endpoints ─────────────────────────────────────────────────────────

// GetUserById handles GET /v1/users/{id}.
func (h *OpenAPIHandler) GetUserById(ctx echo.Context, id string) error {
	user, err := h.userSvc.GetByID(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}
	return ctx.JSON(http.StatusOK, toAPIUserResponse(user))
}

// GetUserByIdInternal handles GET /v1/internal/users/{id}.
func (h *OpenAPIHandler) GetUserByIdInternal(ctx echo.Context, id string) error {
	user, err := h.userSvc.GetByID(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}
	return ctx.JSON(http.StatusOK, toAPIUserResponse(user))
}

// CreateUser handles POST /v1/users.
func (h *OpenAPIHandler) CreateUser(ctx echo.Context) error {
	var req api.CreateUserRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid request body: " + err.Error()})
	}

	// Basic validation
	if req.Email == "" {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "email is required"})
	}
	if req.Name == "" {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "name is required"})
	}

	now := time.Now().UTC()
	domainUser := toDomainUser(&req, uuid.New().String(), now)

	resp, err := h.userSvc.Create(ctx.Request().Context(), domainUser)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusCreated, toAPIUserResponseFromResp(resp))
}

// UpdateUser handles PUT /v1/users/{id}.
func (h *OpenAPIHandler) UpdateUser(ctx echo.Context, id string) error {
	var req api.CreateUserRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid request body: " + err.Error()})
	}

	user, err := h.userSvc.GetByID(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}

	user.Name = req.Name
	if req.Bio != nil {
		user.Bio = *req.Bio
	} else {
		user.Bio = ""
	}
	if req.PhotoUrls != nil {
		user.PhotoURLs = *req.PhotoUrls
	} else {
		user.PhotoURLs = nil
	}
	user.UpdatedAt = time.Now().UTC()

	if err := h.userSvc.Update(ctx.Request().Context(), user); err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, toAPIUserResponse(user))
}

// DeleteUser handles DELETE /v1/users/{id}.
// Uses PostgreSQL RETURNING to delete and return the user data in one atomic query.
func (h *OpenAPIHandler) DeleteUser(ctx echo.Context, id string) error {
	user, err := h.userSvc.Delete(ctx.Request().Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
		}
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, api.DeleteUserResponse{
		Id:      &user.ID,
		Email:   &user.Email,
		Name:    &user.Name,
		Message: strPtr("user deleted successfully"),
	})
}

// ListUsers handles GET /v1/users.
func (h *OpenAPIHandler) ListUsers(ctx echo.Context, params api.ListUsersParams) error {
	page := 1
	limit := 20

	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}
	if params.Limit != nil && *params.Limit > 0 {
		limit = *params.Limit
	}
	if limit > 100 {
		limit = 100
	}

	users, total, err := h.userSvc.List(ctx.Request().Context(), page, limit)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
	}

	userResponses := make([]api.UserResponse, 0, len(users))
	for _, u := range users {
		userResponses = append(userResponses, toAPIUserResponse(u))
	}

	return ctx.JSON(http.StatusOK, api.ListUsersResponse{
		Users: &userResponses,
		Total: &total,
		Page:  &page,
		Limit: &limit,
	})
}

// ─── Auth endpoints ─────────────────────────────────────────────────────────

// GoogleLogin handles GET /v1/auth/google/login.
func (h *OpenAPIHandler) GoogleLogin(ctx echo.Context) error {
	state := generateStateToken()
	url := h.authSvc.GenerateGoogleLoginURL(state)
	return ctx.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback handles GET /v1/auth/google/callback.
func (h *OpenAPIHandler) GoogleCallback(ctx echo.Context, params api.GoogleCallbackParams) error {
	if params.Code == "" {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "missing authorization code"})
	}
	if params.State == "" {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "missing state parameter"})
	}

	authResp, err := h.authSvc.HandleGoogleCallback(ctx.Request().Context(), params.Code)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "authentication failed: " + err.Error()})
	}

	return ctx.JSON(http.StatusOK, toAPIAuthResponse(authResp))
}

// ─── Conversion helpers ─────────────────────────────────────────────────────

func toAPIUserResponse(u *domain.User) api.UserResponse {
	email := u.Email
	name := u.Name
	bio := u.Bio
	avatarURL := u.AvatarURL
	var photoUrls []string
	if u.PhotoURLs != nil {
		photoUrls = make([]string, len(u.PhotoURLs))
		copy(photoUrls, u.PhotoURLs)
	}
	createdAt := u.CreatedAt
	updatedAt := u.UpdatedAt

	return api.UserResponse{
		Id:        &u.ID,
		Email:     &email,
		Name:      &name,
		Bio:       &bio,
		AvatarUrl: &avatarURL,
		PhotoUrls: &photoUrls,
		CreatedAt: &createdAt,
		UpdatedAt: &updatedAt,
	}
}

func toAPIUserResponseFromResp(resp *domain.UserResponse) api.UserResponse {
	return api.UserResponse{
		Id:        &resp.ID,
		Email:     &resp.Email,
		Name:      &resp.Name,
		Bio:       &resp.Bio,
		AvatarUrl: &resp.AvatarURL,
		PhotoUrls: &resp.PhotoURLs,
		CreatedAt: ptrTimeFromStr(resp.CreatedAt),
		UpdatedAt: ptrTimeFromStr(resp.UpdatedAt),
	}
}

func toAPIAuthResponse(authResp *domain.AuthResponse) api.AuthResponse {
	user := authResp.User
	return api.AuthResponse{
		Token: &authResp.Token,
		User: &api.UserProfileResponse{
			Id:        &user.ID,
			Email:     &user.Email,
			Name:      &user.Name,
			AvatarUrl: &user.AvatarURL,
			CreatedAt: ptrTimeFromStr(user.CreatedAt),
		},
	}
}

func toDomainUser(req *api.CreateUserRequest, id string, now time.Time) *domain.User {
	bio := ""
	if req.Bio != nil {
		bio = *req.Bio
	}
	var photoUrls []string
	if req.PhotoUrls != nil {
		photoUrls = *req.PhotoUrls
	}
	return &domain.User{
		ID:        id,
		Email:     string(req.Email),
		Name:      req.Name,
		Bio:       bio,
		PhotoURLs: photoUrls,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func generateStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func strPtr(s string) *string {
	return &s
}

func ptrTimeFromStr(s string) *time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
