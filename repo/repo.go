package repo

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/nikonor/asb/domain"
	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/log"
)

type Memo struct {
	Token     string
	MessageId int
}

type Repo struct {
	logger   log.Logger
	locker   sync.Mutex
	file     string
	users    map[int64]struct{}
	tmpUsers map[int64]Memo
}

func New(ctx context.Context, logger log.Logger) (*Repo, error) {
	m, err := initCache(logger, "./data/users.lst") // TODO: file to cfg
	if err != nil {
		return nil, err
	}
	logger.Debug(ctx, "init cache:"+fmt.Sprintf("%v", m))
	return &Repo{
		file:     "./data/users.lst",
		logger:   logger,
		users:    m,
		tmpUsers: make(map[int64]Memo),
	}, nil

}

func (r *Repo) IsUserExists(ctx context.Context, userId int64) (string, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	r.logger.Debug(context.Background(), "userId="+strconv.FormatInt(userId, 10))

	if _, ok := r.users[userId]; ok {
		return domain.Exist, nil
	}

	if _, ok := r.tmpUsers[userId]; ok {
		return domain.Exist, nil
	}

	r.tmpUsers[userId] = Memo{Token: uuid.NewString()}

	return r.tmpUsers[userId].Token, nil
}

func (r *Repo) ValidateNewUser(ctx context.Context, userId int64, data string) (bool, int, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	u, ok := r.tmpUsers[userId]
	if !ok || (ok && u.Token != data) {
		return false, 0, nil
	}

	if err := r.addUserToFileUnsafe(userId); err != nil {
		r.logger.Warn(ctx, "error on write to file::"+err.Error())
	}
	delete(r.tmpUsers, userId)
	r.users[userId] = struct{}{}

	return true, u.MessageId, nil
}

func (r *Repo) SaveMessageLink(userId int64, messageID int) {
	r.locker.Lock()
	defer r.locker.Unlock()

	obj, ok := r.tmpUsers[userId]
	if !ok {
		return
	}
	obj.MessageId = messageID
	r.tmpUsers[userId] = obj
}

func initCache(logger log.Logger, path string) (map[int64]struct{}, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	m := make(map[int64]struct{})

	scanner := bufio.NewScanner(fh)

	for scanner.Scan() {
		line := scanner.Text()
		logger.Debug(context.Background(), "line: ["+line+"]")
		userId, err := strconv.ParseInt(strings.TrimSpace(line), 10, 64)
		if err != nil {
			return nil, err
		}
		m[userId] = struct{}{}
	}

	return m, nil
}

func (r *Repo) addUserToFileUnsafe(userId int64) error {
	fh, err := os.OpenFile(r.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.WithMessage(err, "open file")
	}
	defer fh.Close()

	// if _, err = fh.Seek(0, 2); err != nil {
	// 	return errors.WithMessage(err, "seek")
	// }
	if _, err = fh.WriteString(fmt.Sprintf("%d\n", userId)); err != nil {
		return err
	}

	return nil
}
