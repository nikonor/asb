package repo

type Repo struct{}

func New() Repo {
	return Repo{}
}

func (r Repo) IsExistUser(from int64) (bool, error) {
	return true, nil
}
