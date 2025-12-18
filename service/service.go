package service

import (
	"context"

	"github.com/txix-open/isp-kit/log"
)

type Repo interface {
	GetUserStatus(ctx context.Context, userId int64) string
	ValidateNewUser(ctx context.Context, userId int64, data string) (bool, int)
	SaveMessageLink(userId int64, messageID int)
	SaveToPending(ctx context.Context, chatId int64, userId int64, messageId int) error
	SaveValidUser(ctx context.Context, userId int64) error
	SaveUserToBlackList(ctx context.Context, userId int64) error
}

type Service struct {
	logger log.Logger
	repo   Repo
}

func (s *Service) HandleCallback(ctx context.Context, userId int64, data string) int {
	ok, botMessageId := s.repo.ValidateNewUser(ctx, userId, data)
	if ok {
		if err := s.repo.SaveValidUser(ctx, userId); err != nil {
			s.logger.Warn(ctx, "SaveValidUser::"+err.Error())
		}
		return botMessageId
	}
	if err := s.repo.SaveUserToBlackList(ctx, userId); err != nil {
		s.logger.Warn(ctx, "SaveUserToBlackList::"+err.Error())
	}

	return botMessageId
}

func New(logger log.Logger, repo Repo) *Service {
	return &Service{logger: logger, repo: repo}
}

func (s *Service) GetUserStatus(ctx context.Context, userId int64) string {
	return s.repo.GetUserStatus(ctx, userId)
}

func (s *Service) SaveMessageLink(userId int64, messageID int) {
	s.repo.SaveMessageLink(userId, messageID)
}

func (s *Service) SaveToPending(ctx context.Context, chatId int64, userId int64, messageId int) error {
	return s.repo.SaveToPending(ctx, chatId, userId, messageId)
}
