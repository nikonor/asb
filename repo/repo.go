package repo

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/nikonor/asb/domain"
	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/log"
)

type Memo struct {
	Token        string
	BotMessageId int
}

type TaskForDelete struct {
	ChatId    int64
	UserId    int64
	MessageId int
	TS        time.Time
}

type Repo struct {
	logger         log.Logger
	locker         sync.Mutex
	path           string
	users          map[int64]struct{}
	blacklist      map[int64]struct{}
	tmpUsers       map[int64]Memo
	queryForDelete []TaskForDelete
	senderChan     chan domain.SendObject
}

func New(ctx context.Context, logger log.Logger, senderChan chan domain.SendObject) (*Repo, error) {
	m, b, err := initRepo(logger, "./data") // TODO: path to cfg
	if err != nil {
		return nil, err
	}
	logger.Debug(ctx, "init cache:"+fmt.Sprintf("%v", m))
	r := Repo{
		path:           "./data",
		logger:         logger,
		users:          m,
		blacklist:      b,
		tmpUsers:       make(map[int64]Memo),
		queryForDelete: make([]TaskForDelete, 0),
		senderChan:     senderChan,
	}

	go r.bg(ctx)

	return &r, nil

}

func (r *Repo) IsUserExists(ctx context.Context, userId int64) (string, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	r.logger.Debug(context.Background(), "userId="+strconv.FormatInt(userId, 10))

	if _, ok := r.users[userId]; ok {
		return domain.Exist, nil
	}

	if _, ok := r.tmpUsers[userId]; ok {
		return domain.TmpUser, nil
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
		r.logger.Warn(ctx, "error on write to path::"+err.Error())
	}
	delete(r.tmpUsers, userId)
	r.users[userId] = struct{}{}
	if err := os.Remove("./data/tmp/" + strconv.FormatInt(userId, 10)); err != nil {
		r.logger.Warn(ctx, "error on delete to path::"+err.Error())
	}

	return true, u.BotMessageId, nil
}

func (r *Repo) SaveMessageLink(userId int64, messageID int) {
	r.locker.Lock()
	defer r.locker.Unlock()

	obj, ok := r.tmpUsers[userId]
	if !ok {
		return
	}
	obj.BotMessageId = messageID
	r.tmpUsers[userId] = obj
}

func (r *Repo) SaveToQuery(ctx context.Context, chatId int64, userId int64, messageId int) error {
	fh, err := os.OpenFile("./data/tmp/"+strconv.FormatInt(userId, 10), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.WithMessage(err, "open path")
	}
	defer func() { _ = fh.Close() }()

	if _, err = fh.WriteString(strconv.Itoa(messageId)); err != nil {
		return errors.WithMessage(err, "write to path")
	}

	r.locker.Lock()
	defer r.locker.Unlock()
	r.queryForDelete = append(r.queryForDelete, TaskForDelete{
		ChatId:    chatId,
		UserId:    userId,
		MessageId: messageId,
		TS:        time.Now().Add(15 * time.Second), // TODO: cfg
	})

	return nil
}

func initRepo(logger log.Logger, path string) (map[int64]struct{}, map[int64]struct{}, error) {
	fh, err := os.OpenFile(path+"/users.lst", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "open user.lst")
	}
	defer func() { _ = fh.Close() }()

	m := make(map[int64]struct{})

	scanner := bufio.NewScanner(fh)

	for scanner.Scan() {
		line := scanner.Text()
		logger.Debug(context.Background(), "line: ["+line+"]")
		userId, err := strconv.ParseInt(strings.TrimSpace(line), 10, 64)
		if err != nil {
			return nil, nil, errors.WithMessage(err, "parse user.lst")
		}
		m[userId] = struct{}{}
	}

	fh2, err := os.OpenFile(path+"/black.lst", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "open black.lst")
	}
	defer func() { _ = fh2.Close() }()

	b := make(map[int64]struct{})

	scanner = bufio.NewScanner(fh2)

	for scanner.Scan() {
		line := scanner.Text()
		logger.Debug(context.Background(), "line: ["+line+"]")
		userId, err := strconv.ParseInt(strings.TrimSpace(line), 10, 64)
		if err != nil {
			return nil, nil, errors.WithMessage(err, "parse black.lst")
		}
		b[userId] = struct{}{}
	}

	return m, b, nil
}

func (r *Repo) bg(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.logger.Debug(ctx, "context done")
			return
		case <-time.After(time.Second):
			r.checkQuery(ctx)
		}
	}
}

func (r *Repo) addUserToFileUnsafe(userId int64) error {
	fh, err := os.OpenFile(r.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.WithMessage(err, "open path")
	}
	defer func() { _ = fh.Close() }()

	if _, err = fh.WriteString(fmt.Sprintf("%d\n", userId)); err != nil {
		return err
	}

	return nil
}

func (r *Repo) checkQuery(ctx context.Context) {
	r.locker.Lock()
	defer r.locker.Unlock()
	now := time.Now()

	var (
		task TaskForDelete
	)

	for _, task = range r.queryForDelete {
		r.logger.Debug(ctx, fmt.Sprintf("%s", task.TS.String()))
		if task.TS.After(now) {
			break
		}
		r.senderChan <- domain.SendObject{Msg: tgbotapi.DeleteMessageConfig{
			ChatID:    task.ChatId,
			MessageID: task.MessageId,
		}}
		r.senderChan <- domain.SendObject{Msg: tgbotapi.DeleteMessageConfig{
			ChatID:    task.ChatId,
			MessageID: r.tmpUsers[task.UserId].BotMessageId,
		}}

		// TODO: бан юзера

		delete(r.tmpUsers, task.UserId)

		r.queryForDelete = r.queryForDelete[1:]
	}
}
