package grpc_server

import (
	"context"
	"errors"
	"fmt"

	"coder_manager/internal/coder_service"
	"coder_manager/internal/repo"
	"coder_manager/pkg/proto"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CoderManagerServer struct {
	service *coder_service.Service
	proto.UnimplementedCoderManagerServiceServer
}

func NewCoderManagerServer(service *coder_service.Service) *CoderManagerServer {
	return &CoderManagerServer{service: service}
}

func (s *CoderManagerServer) CreateEditorSession(ctx context.Context, req *proto.CreateEditorSessionRequest) (*proto.CreateEditorSessionResponse, error) {
	response, err := s.service.CreateEditorSession(ctx, coder_service.CreateEditorSessionRequest{
		Repo:       req.GetRepo(),
		Branch:     req.GetBranch(),
		Path:       req.GetPath(),
		ChatID:     req.GetChatId(),
		TTLSeconds: req.GetTtlSeconds(),
	})
	if err != nil {
		zap.S().Errorw("create editor session failed", "error", err)
		return nil, convertError(err)
	}
	var expiresAt *timestamppb.Timestamp
	if response.ExpiresAt != nil {
		expiresAt = timestamppb.New(*response.ExpiresAt)
	}
	return &proto.CreateEditorSessionResponse{
		OneTimeUrl: response.OneTimeURL,
		SessionId:  stringID(response.SessionID),
		ExpiresAt:  expiresAt,
	}, nil
}

func convertError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repo.ErrUserNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, repo.ErrTokenNotFound) {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	if errors.Is(err, coder_service.ErrInvalidRequest) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}

func stringID(value int64) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%d", value)
}
