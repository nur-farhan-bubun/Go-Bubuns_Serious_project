package handler

import (
	"context"

	sharedUser "github.com/ride-sharing/shared/proto/user"
	"github.com/ride-sharing/user-service/internal/service"
)

type GrpcHandler struct {
	sharedUser.UnimplementedUserServiceServer
	svc *service.Service
}

func NewGrpcHandler(svc *service.Service) *GrpcHandler {
	return &GrpcHandler{svc: svc}
}

func (h *GrpcHandler) GetInternalProfile(ctx context.Context, req *sharedUser.InternalProfileRequest) (*sharedUser.InternalProfileResponse, error) {
	user, err := h.svc.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	profile, _ := h.svc.GetProfile(ctx, req.UserId)

	resp := &sharedUser.InternalProfileResponse{
		Id:   user.ID,
		IsActive: true,
	}
	if profile != nil {
		resp.DisplayName = profile.DisplayName
		resp.AvatarUrl = profile.AvatarURL
	}
	return resp, nil
}

func (h *GrpcHandler) CheckBlockStatus(ctx context.Context, req *sharedUser.BlocKStatusCheckRequest) (*sharedUser.BlocKStatusCheckResponse, error) {
	isBlocked, err := h.svc.CheckBlockStatus(ctx, req.SenderId, req.RecId)
	if err != nil {
		return nil, err
	}
	return &sharedUser.BlocKStatusCheckResponse{IsBlocked: isBlocked}, nil
}
