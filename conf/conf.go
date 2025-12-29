package conf

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/txix-open/isp-kit/json"
	"github.com/txix-open/isp-kit/validator"
)

type Config struct {
	DataPath               string `json:"data_path" validate:"required"`
	SenderWorkers          int    `json:"sender_workers"  validate:"required"`
	ReceiverWorkers        int    `json:"receiver_workers"  validate:"required"`
	TlgTimeout             int    `json:"tlg_timeout"  validate:"required"`
	WorkerTimeoutInSeconds int    `json:"worker_timeout_in_seconds"  validate:"required"`
	UserTimeoutInSeconds   int    `json:"user_timeout_in_seconds"  validate:"required"`
	Messages               struct {
		Welcome    string `json:"welcome"  validate:"required"`
		ButtonText string `json:"button_text"  validate:"required"`
		Ask        string `json:"ask"  validate:"required"`
	} `json:"messages"  validate:"required"`
}

func ReadConfig(fileName string) (*Config, error) {
	fh, err := os.Open(fileName)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to open file")
	}
	defer fh.Close()

	var cfg Config

	decoder := json.NewDecoder(fh)
	if err = decoder.Decode(&cfg); err != nil {
		return nil, errors.WithMessage(err, "failed to parse file")
	}

	val := validator.New()
	if ok, details := val.Validate(&cfg); !ok {
		return nil, errors.New(fmt.Sprintf("%s is not valide: %v", fileName, details))
	}

	return &cfg, nil
}
