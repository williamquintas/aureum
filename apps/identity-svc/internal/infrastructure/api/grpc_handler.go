package api

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/aureum/identity-svc/internal/application"
	"github.com/aureum/identity-svc/internal/domain"
	"github.com/aureum/pkg/telemetry"
	identityv1 "github.com/aureum/proto/gen/identity/identityv1"
)

type GRPCHandler struct {
	identityv1.UnimplementedIdentityServiceServer
	authSvc  *application.AuthService
	authzSvc *application.AuthorizationService
}

func NewGRPCHandler(authSvc *application.AuthService, authzSvc *application.AuthorizationService) *GRPCHandler {
	return &GRPCHandler{authSvc: authSvc, authzSvc: authzSvc}
}

func (h *GRPCHandler) ValidateToken(
	ctx context.Context, req *identityv1.ValidateTokenRequest,
) (*identityv1.ValidateTokenResponse, error) {
	start := time.Now()
	user, err := h.authSvc.ValidateToken(ctx, req.Token)
	if err != nil {
		return &identityv1.ValidateTokenResponse{Valid: false}, nil
	}

	telemetry.RecordRequest(ctx, "validate_token", "200", time.Since(start))
	return &identityv1.ValidateTokenResponse{
		Valid:  true,
		UserId: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		Roles:  user.Roles,
	}, nil
}

func (h *GRPCHandler) GetUser(
	ctx context.Context, req *identityv1.GetUserRequest,
) (*identityv1.GetUserResponse, error) {
	start := time.Now()
	profile, err := h.authSvc.GetProfile(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	telemetry.RecordRequest(ctx, "get_user", "200", time.Since(start))
	resp := &identityv1.GetUserResponse{
		UserId:        profile.ID,
		Email:         profile.Email,
		EmailVerified: profile.EmailVerified,
		Name:          profile.Name,
		Status:        profile.Status,
		MfaEnabled:    profile.MFAEnabled,
		Roles:         profile.Roles,
		CreatedAt:     timestamppb.New(profile.CreatedAt),
		UpdatedAt:     timestamppb.New(profile.UpdatedAt),
	}
	if profile.AvatarURL != "" {
		resp.AvatarUrl = profile.AvatarURL
	}
	return resp, nil
}

func (h *GRPCHandler) ABACCheck(
	ctx context.Context, req *identityv1.ABACCheckRequest,
) (*identityv1.ABACCheckResponse, error) {
	start := time.Now()
	resp, err := h.authzSvc.Evaluate(ctx, application.ABACCheckRequest{
		UserID:          req.UserId,
		ResourceType:    req.ResourceType,
		ResourceID:      req.ResourceId,
		Action:          req.Action,
		ResourceOwnerID: req.ResourceOwnerId,
		Attributes:      req.Attributes,
	})
	if err != nil {
		return &identityv1.ABACCheckResponse{Allowed: false, Reason: "internal error"}, nil
	}

	telemetry.RecordRequest(ctx, "abac_check", "200", time.Since(start))
	return &identityv1.ABACCheckResponse{
		Allowed: resp.Allowed,
		Reason:  resp.Reason,
	}, nil
}
