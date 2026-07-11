package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
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
	profile, _ := h.userSvc.GetProfile(ctx.Request().Context(), id)
	return ctx.JSON(http.StatusOK, toAPIUserResponse(user, profile))
}

// GetUserByIdInternal handles GET /v1/internal/users/{id}.
func (h *OpenAPIHandler) GetUserByIdInternal(ctx echo.Context, id string) error {
	user, err := h.userSvc.GetByID(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}
	profile, _ := h.userSvc.GetProfile(ctx.Request().Context(), id)
	return ctx.JSON(http.StatusOK, toAPIUserResponse(user, profile))
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

	now := time.Now().UTC()
	userID := uuid.New().String()
	domainUser := toDomainUser(&req, userID, now)

	displayName := ""
	avatarURL := ""
	if req.DisplayName != nil && *req.DisplayName != "" {
		displayName = *req.DisplayName
	}
	if req.AvatarUrl != nil && *req.AvatarUrl != "" {
		avatarURL = *req.AvatarUrl
	}

	resp, err := h.userSvc.Create(ctx.Request().Context(), domainUser, displayName, avatarURL)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusCreated, toAPIUserResponseFromResp(resp))
}

// UpdateUser handles PUT /v1/users/{id}.
func (h *OpenAPIHandler) UpdateUser(ctx echo.Context, id string) error {
	var req api.UpdateUserRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid request body: " + err.Error()})
	}

	user, err := h.userSvc.GetByID(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}

	if req.Email != nil {
		user.Email = string(*req.Email)
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	user.UpdatedAt = time.Now().UTC()

	if err := h.userSvc.Update(ctx.Request().Context(), user); err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
	}

	profile, _ := h.userSvc.GetProfile(ctx.Request().Context(), id)
	return ctx.JSON(http.StatusOK, toAPIUserResponse(user, profile))
}

// DeleteUser handles DELETE /v1/users/{id}.
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
		userResponses = append(userResponses, toAPIUserResponse(u, nil))
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

// ─── Universal Profile endpoints ────────────────────────────────────────────

// GetProfile handles GET /v1/users/{id}/profile.
func (h *OpenAPIHandler) GetProfile(ctx echo.Context, id string) error {
	profile, err := h.userSvc.GetProfile(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}
	return ctx.JSON(http.StatusOK, toAPIProfileResponse(profile))
}

// UpdateProfile handles PUT /v1/users/{id}/profile.
func (h *OpenAPIHandler) UpdateProfile(ctx echo.Context, id string) error {
	var req api.ProfileRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid request body: " + err.Error()})
	}

	profile := toDomainProfile(&req, id)
	if err := h.userSvc.UpdateProfile(ctx.Request().Context(), profile); err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, toAPIProfileResponse(profile))
}

// ─── Dating Profile endpoints ───────────────────────────────────────────────

// GetDatingProfile handles GET /v1/users/{id}/dating-profile.
func (h *OpenAPIHandler) GetDatingProfile(ctx echo.Context, id string) error {
	profile, err := h.userSvc.GetDatingProfile(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}
	return ctx.JSON(http.StatusOK, toAPIDatingProfileResponse(profile))
}

// UpdateDatingProfile handles PUT /v1/users/{id}/dating-profile.
func (h *OpenAPIHandler) UpdateDatingProfile(ctx echo.Context, id string) error {
	var req api.DatingProfileRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid request body: " + err.Error()})
	}

	profile := toDomainDatingProfile(&req, id)
	if err := h.userSvc.UpdateDatingProfile(ctx.Request().Context(), profile); err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, toAPIDatingProfileResponse(profile))
}

// DeleteDatingProfile handles DELETE /v1/users/{id}/dating-profile.
func (h *OpenAPIHandler) DeleteDatingProfile(ctx echo.Context, id string) error {
	if err := h.userSvc.DeleteDatingProfile(ctx.Request().Context(), id); err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}
	return ctx.NoContent(http.StatusNoContent)
}

// ─── Worker Profile endpoints ───────────────────────────────────────────────

// GetWorkerProfile handles GET /v1/users/{id}/worker-profile.
func (h *OpenAPIHandler) GetWorkerProfile(ctx echo.Context, id string) error {
	profile, err := h.userSvc.GetWorkerProfile(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}
	return ctx.JSON(http.StatusOK, toAPIWorkerProfileResponse(profile))
}

// UpdateWorkerProfile handles PUT /v1/users/{id}/worker-profile.
func (h *OpenAPIHandler) UpdateWorkerProfile(ctx echo.Context, id string) error {
	var req api.WorkerProfileRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid request body: " + err.Error()})
	}

	profile := toDomainWorkerProfile(&req, id)
	if err := h.userSvc.UpdateWorkerProfile(ctx.Request().Context(), profile); err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusOK, toAPIWorkerProfileResponse(profile))
}

// DeleteWorkerProfile handles DELETE /v1/users/{id}/worker-profile.
func (h *OpenAPIHandler) DeleteWorkerProfile(ctx echo.Context, id string) error {
	if err := h.userSvc.DeleteWorkerProfile(ctx.Request().Context(), id); err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}
	return ctx.NoContent(http.StatusNoContent)
}

// ─── Profile Photo endpoints ────────────────────────────────────────────────

// ListPhotos handles GET /v1/users/{id}/photos.
func (h *OpenAPIHandler) ListPhotos(ctx echo.Context, id string) error {
	photos, err := h.userSvc.ListPhotos(ctx.Request().Context(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}
	return ctx.JSON(http.StatusOK, toAPIPhotoListResponse(photos))
}

// AddPhoto handles POST /v1/users/{id}/photos.
func (h *OpenAPIHandler) AddPhoto(ctx echo.Context, id string) error {
	var req api.AddPhotoRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid request body: " + err.Error()})
	}

	photo := toDomainPhoto(&req, id)
	if err := h.userSvc.AddPhoto(ctx.Request().Context(), photo); err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: err.Error()})
	}

	return ctx.JSON(http.StatusCreated, toAPIPhotoResponse(photo))
}

// DeletePhoto handles DELETE /v1/users/{id}/photos/{photoId}.
func (h *OpenAPIHandler) DeletePhoto(ctx echo.Context, id string, photoId openapi_types.UUID) error {
	if err := h.userSvc.DeletePhoto(ctx.Request().Context(), photoId.String()); err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}
	return ctx.NoContent(http.StatusNoContent)
}

// SetPrimaryPhoto handles PUT /v1/users/{id}/photos/{photoId}/primary.
func (h *OpenAPIHandler) SetPrimaryPhoto(ctx echo.Context, id string, photoId openapi_types.UUID) error {
	photo, err := h.userSvc.SetPrimaryPhoto(ctx.Request().Context(), photoId.String(), id)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, api.ErrorResponse{Error: err.Error()})
	}
	return ctx.JSON(http.StatusOK, toAPIPhotoResponse(photo))
}

// ─── Conversion helpers ─────────────────────────────────────────────────────

func toAPIUserResponse(u *domain.User, profile *domain.Profile) api.UserResponse {
	email := u.Email
	createdAt := u.CreatedAt
	updatedAt := u.UpdatedAt
	role := api.UserResponseRole(u.Role)

	var profileResp *api.ProfileResponse
	if profile != nil {
		p := toAPIProfileResponse(profile)
		profileResp = &p
	}

	return api.UserResponse{
		Id:          &u.ID,
		Email:       &email,
		Phone:       &u.Phone,
		Role:        &role,
		IsOnboarded: &u.IsOnboarded,
		Profile:     profileResp,
		CreatedAt:   &createdAt,
		UpdatedAt:   &updatedAt,
	}
}

func toAPIUserResponseFromResp(resp *domain.UserResponse) api.UserResponse {
	return api.UserResponse{
		Id:        &resp.ID,
		Email:     &resp.Email,
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
	phone := ""
	if req.Phone != nil {
		phone = *req.Phone
	}
	return &domain.User{
		ID:        id,
		Email:     string(req.Email),
		Phone:     phone,
		Role:      domain.UserRoleSocialOnly,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func toAPIProfileResponse(p *domain.Profile) api.ProfileResponse {
	return api.ProfileResponse{
		Id:          &p.ID,
		UserId:      &p.UserID,
		DisplayName: &p.DisplayName,
		AvatarUrl:   &p.AvatarURL,
		Bio:         &p.Bio,
		UpdatedAt:   &p.UpdatedAt,
	}
}

func toDomainProfile(req *api.ProfileRequest, userID string) *domain.Profile {
	profile := &domain.Profile{
		UserID: userID,
	}
	if req.DisplayName != nil {
		profile.DisplayName = *req.DisplayName
	}
	if req.AvatarUrl != nil {
		profile.AvatarURL = *req.AvatarUrl
	}
	if req.Bio != nil {
		profile.Bio = *req.Bio
	}
	return profile
}

// ─── Dating Profile conversion helpers ──────────────────────────────────────

func toAPIDatingProfileResponse(p *domain.DatingProfile) api.DatingProfileResponse {
	birthDate := openapi_types.Date{Time: p.BirthDate}
	return api.DatingProfileResponse{
		Gender:           &p.Gender,
		InterestedIn:     &p.InterestedIn,
		BirthDate:        &birthDate,
		HeightCm:         p.HeightCm,
		RelationshipGoal: strPtrOrNil(p.RelationshipGoal),
		UpdatedAt:        &p.UpdatedAt,
	}
}

func toDomainDatingProfile(req *api.DatingProfileRequest, userID string) *domain.DatingProfile {
	profile := &domain.DatingProfile{
		UserID:       userID,
		Gender:       string(req.Gender),
		InterestedIn: string(req.InterestedIn),
		BirthDate:    req.BirthDate.Time,
	}
	if req.HeightCm != nil {
		profile.HeightCm = req.HeightCm
	}
	if req.RelationshipGoal != nil {
		profile.RelationshipGoal = string(*req.RelationshipGoal)
	}
	return profile
}

// ─── Worker Profile conversion helpers ──────────────────────────────────────

func toAPIWorkerProfileResponse(p *domain.WorkerProfile) api.WorkerProfileResponse {
	var hourlyRate *float32
	if p.HourlyRate != nil {
		hr := float32(*p.HourlyRate)
		hourlyRate = &hr
	}
	return api.WorkerProfileResponse{
		Skills:             &p.Skills,
		HourlyRate:         hourlyRate,
		IsAvailable:        &p.IsAvailable,
		CompletedJobsCount: &p.CompletedJobsCount,
		RatingAvg:          float32Ptr(p.RatingAvg),
		UpdatedAt:          &p.UpdatedAt,
	}
}

func toDomainWorkerProfile(req *api.WorkerProfileRequest, userID string) *domain.WorkerProfile {
	profile := &domain.WorkerProfile{
		UserID: userID,
	}
	if req.Skills != nil {
		profile.Skills = *req.Skills
	}
	if req.HourlyRate != nil {
		hr := float64(*req.HourlyRate)
		profile.HourlyRate = &hr
	}
	if req.IsAvailable != nil {
		profile.IsAvailable = *req.IsAvailable
	}
	return profile
}

// ─── Photo conversion helpers ───────────────────────────────────────────────

func toAPIPhotoResponse(p *domain.ProfilePhoto) api.PhotoResponse {
	photoID := uuid.MustParse(p.ID)
	return api.PhotoResponse{
		Id:        &photoID,
		S3Url:     &p.S3URL,
		SortOrder: &p.SortOrder,
		IsPrimary: &p.IsPrimary,
		CreatedAt: &p.CreatedAt,
	}
}

func toAPIPhotoListResponse(photos []*domain.ProfilePhoto) []api.PhotoResponse {
	result := make([]api.PhotoResponse, 0, len(photos))
	for _, p := range photos {
		result = append(result, toAPIPhotoResponse(p))
	}
	return result
}

func toDomainPhoto(req *api.AddPhotoRequest, userID string) *domain.ProfilePhoto {
	photo := &domain.ProfilePhoto{
		UserID: userID,
		S3URL:  req.S3Url,
	}
	if req.IsPrimary != nil {
		photo.IsPrimary = *req.IsPrimary
	}
	return photo
}

func generateStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func strPtr(s string) *string {
	return &s
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func float32Ptr(f float64) *float32 {
	v := float32(f)
	return &v
}

func ptrTimeFromStr(s string) *time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
