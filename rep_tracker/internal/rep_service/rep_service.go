package rep_service

import (
	"context"
	"rep_tracker/internal/repo"
	"rep_tracker/internal/server_model"
	"rep_tracker/pkg/errs"
	"rep_tracker/pkg/github"
)

type RepService struct {
	ghClient   *github.GithubClient
	tokenRepo  repo.TokenRepo
	serverRepo repo.ServerRepo
}

func NewRepService(ghClient *github.GithubClient, tokenRepo repo.TokenRepo, serverRepo repo.ServerRepo) *RepService {
	return &RepService{}
}

func (service *RepService) AddTrackingRepo(ctx context.Context, trackingRepo *server_model.TrackingRepo) error {
	token, err := service.tokenRepo.GetToken(ctx, trackingRepo.ChatID)
	if err != nil {
		return errs.ErrInternal
	}
	exists, err := service.ghClient.CheckRepo(ctx, token, trackingRepo.Link)
	if err != nil {
		return err
	}
	if !exists {
		return errs.ErrRepoNotFound
	}
	return service.serverRepo.AddNotificationRep(ctx, trackingRepo)
}

func (service *RepService) RemoveTrackingRepo(ctx context.Context, trackingRepo *server_model.TrackingRepo) error {
	return service.serverRepo.RemoveNotificationRep(ctx, trackingRepo)
}
