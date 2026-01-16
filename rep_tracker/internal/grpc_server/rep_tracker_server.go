package grpc_server

import (
	"context"
	"rep_tracker/internal/rep_service"
	"rep_tracker/internal/server_model"
	"rep_tracker/pkg/errs"
	"rep_tracker/pkg/proto"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	USER_NOT_FOUND_REASON = "USER_NOT_FOUND_REASON"
	REPO_NOT_FOUND_REASON = "REPO_NOT_FOUND_REASON"
)

type RepTrackerServiceServer struct {
	repService *rep_service.RepService
	proto.UnimplementedRepTrackerServiceServer
}

func NewRepTrackerServiceServer(repService *rep_service.RepService) *RepTrackerServiceServer {
	return &RepTrackerServiceServer{repService: repService}
}

func (server *RepTrackerServiceServer) AddTrackingRepo(ctx context.Context, trackingRepo *proto.TrackingRepo) (*emptypb.Empty, error) {
	return server.doWithServerModelTrackingRepo(ctx, trackingRepo, server.repService.AddTrackingRepo)

}

func (server *RepTrackerServiceServer) RemoveTrackingRepo(ctx context.Context, trackingRepo *proto.TrackingRepo) (*emptypb.Empty, error) {
	return server.doWithServerModelTrackingRepo(ctx, trackingRepo, server.repService.RemoveTrackingRepo)
}

func (server *RepTrackerServiceServer) doWithServerModelTrackingRepo(ctx context.Context, trackingRepo *proto.TrackingRepo, operation func(context.Context, *server_model.TrackingRepo) error) (*emptypb.Empty, error) {
	modelTrackingRepo, err := parseProtoTrackingRepo(trackingRepo)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &emptypb.Empty{}, convertErrToGrpcError(operation(ctx, modelTrackingRepo))
}

func parseProtoTrackingRepo(trackingRepo *proto.TrackingRepo) (*server_model.TrackingRepo, error) {
	link := trackingRepo.GetLink()
	if link == "" {
		return nil, errs.ErrNotValidData
	}
	chatId := trackingRepo.GetChatId()
	if chatId == "" {
		return nil, errs.ErrNotValidData
	}
	return &server_model.TrackingRepo{Link: link, ChatID: chatId}, nil
}

func convertErrToGrpcError(err error) error {
	if err != nil {
		switch err {
		case errs.ErrUserNotFound:
			st := status.New(codes.NotFound, err.Error())
			detail := &errdetails.ErrorInfo{
				Reason: USER_NOT_FOUND_REASON,
			}
			stWithDetails, _ := st.WithDetails(detail)
			return stWithDetails.Err()
		case errs.ErrNotValidData:
			return status.Error(codes.InvalidArgument, err.Error())
		case errs.ErrRepoNotFound:
			st := status.New(codes.NotFound, err.Error())
			detail := &errdetails.ErrorInfo{
				Reason: REPO_NOT_FOUND_REASON,
			}
			stWithDetails, _ := st.WithDetails(detail)
			return stWithDetails.Err()
		case errs.ErrInvalidToken:
			return status.Error(codes.PermissionDenied, err.Error())
		case errs.ErrInternal:
			return status.Error(codes.Internal, err.Error())
		default:
			return status.Error(codes.Internal, err.Error())
		}
	}
	return nil
}
