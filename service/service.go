package service

import (
	"context"

	"github.com/txix-open/isp-kit/log"
)

type Repo interface {
	IsUserExists(ctx context.Context, userId int64) (string, error)
	ValidateNewUser(ctx context.Context, userId int64, data string) (bool, int, error)
	SaveMessageLink(userId int64, messageID int)
	SaveToQuery(ctx context.Context, chatId int64, userId int64, messageId int) error
}

type Service struct {
	logger log.Logger
	repo   Repo
}

func New(logger log.Logger, repo Repo) *Service {
	return &Service{logger: logger, repo: repo}
}

func (s *Service) IsUserExists(ctx context.Context, userId int64) (string, error) {
	return s.repo.IsUserExists(ctx, userId)
}

func (s *Service) ValidateNewUser(ctx context.Context, userId int64, data string) (bool, int, error) {
	return s.repo.ValidateNewUser(ctx, userId, data)
}

func (s *Service) SaveMessageLink(userId int64, messageID int) {
	s.repo.SaveMessageLink(userId, messageID)
}

func (s *Service) SaveToQuery(ctx context.Context, chatId int64, userId int64, messageId int) error {
	return s.repo.SaveToQuery(ctx, chatId, userId, messageId)
}
