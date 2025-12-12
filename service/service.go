package service

type Repo interface {
	IsExistUser(from int64) (bool, error)
}

type Service struct {
	repo Repo
}

func New(repo Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) IsExistUser(from int64) (bool, error) {
	return s.repo.IsExistUser(from)
}
