package usecase

import (
	"time"

	"go.uber.org/zap"
)

type Usecase struct {
	storage   Storage
	cache     TaskListCache
	log       *zap.Logger
	jwtSecret string
	jwtTTL    time.Duration
}

func New(storage Storage, cache TaskListCache, log *zap.Logger, jwtSecret string, jwtTTL time.Duration) *Usecase {
	return &Usecase{storage: storage, cache: cache, log: log, jwtSecret: jwtSecret, jwtTTL: jwtTTL}
}
