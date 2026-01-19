package rep_service

import (
	"context"
	"rep_tracker/internal/repo"
	"rep_tracker/internal/server_model"
	"rep_tracker/pkg/errs"
	"rep_tracker/pkg/github"
	"go.uber.org/zap"
)

type RepService struct {
	ghClient   *github.GithubClient
	tokenRepo  repo.TokenRepo
	serverRepo repo.ServerRepo
}

func NewRepService(ghClient *github.GithubClient, tokenRepo repo.TokenRepo, serverRepo repo.ServerRepo) *RepService {
	return &RepService{
		ghClient:   ghClient,
		tokenRepo:  tokenRepo,
		serverRepo: serverRepo,
	}
}

func (service *RepService) AddTrackingRepo(ctx context.Context, trackingRepo *server_model.TrackingRepo) error {
	zap.L().Info("Starting AddTrackingRepo in RepService", 
		zap.String("link", trackingRepo.Link), 
		zap.String("chatId", trackingRepo.ChatID))
	
	// Step 1: Get user token
	zap.L().Debug("Step 1: Getting token for chatId", zap.String("chatId", trackingRepo.ChatID))
	token, err := service.tokenRepo.GetToken(ctx, trackingRepo.ChatID)
	if err != nil {
		zap.L().Error("Failed to get token", 
			zap.String("chatId", trackingRepo.ChatID), 
			zap.Error(err))
		return errs.ErrInternal
	}
	zap.L().Debug("Token retrieved successfully", zap.String("chatId", trackingRepo.ChatID))
	
	// Step 2: Check if repository exists on GitHub
	zap.L().Debug("Step 2: Checking repository existence on GitHub", 
		zap.String("link", trackingRepo.Link), 
		zap.String("chatId", trackingRepo.ChatID))
	exists, err := service.ghClient.CheckRepo(ctx, token, trackingRepo.Link)
	if err != nil {
		zap.L().Error("GitHub repo check failed", 
			zap.String("link", trackingRepo.Link), 
			zap.String("chatId", trackingRepo.ChatID),
			zap.Error(err))
		return err
	}
	
	if !exists {
		zap.L().Warn("Repository not found on GitHub", 
			zap.String("link", trackingRepo.Link), 
			zap.String("chatId", trackingRepo.ChatID))
		return errs.ErrRepoNotFound
	}
	zap.L().Info("Repository exists on GitHub", 
		zap.String("link", trackingRepo.Link), 
		zap.String("chatId", trackingRepo.ChatID))
	
	// Step 3: Add to server repository for notifications
	zap.L().Debug("Step 3: Adding repository to server notifications", 
		zap.String("link", trackingRepo.Link), 
		zap.String("chatId", trackingRepo.ChatID))
	err = service.serverRepo.AddNotificationRep(ctx, trackingRepo)
	if err != nil {
		zap.L().Error("Failed to add repository to server notifications", 
			zap.String("link", trackingRepo.Link), 
			zap.String("chatId", trackingRepo.ChatID),
			zap.Error(err))
		return err
	}
	
	zap.L().Info("AddTrackingRepo completed successfully", 
		zap.String("link", trackingRepo.Link), 
		zap.String("chatId", trackingRepo.ChatID))
	
	return nil
}

func (service *RepService) RemoveTrackingRepo(ctx context.Context, trackingRepo *server_model.TrackingRepo) error {
	return service.serverRepo.RemoveNotificationRep(ctx, trackingRepo)
}
