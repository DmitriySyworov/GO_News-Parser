package outbox

import (
	"context"
	"shared/loggers"
	"shared/shconstant"
	"shared/storage"
)

type Outbox struct {
	SharedRedis *storage.Redis
	Logger      *loggers.Logger
}

func NewOutbox(shRedis *storage.Redis, logger *loggers.Logger) *Outbox {
	return &Outbox{
		SharedRedis: shRedis,
		Logger:      logger,
	}
}

func (o *Outbox) AddRedisOutbox(key string, event []byte) error {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), shconstant.CtxTimeoutRedis)
	defer cancel()
	if errLPush := o.SharedRedis.LPush(ctxTimeout, key, event).Err(); errLPush != nil {
		o.Logger.Error("failed to add outbox security event: " + errLPush.Error())
		return errLPush
	}
	return nil
}
func (o *Outbox) RemRangeRedisOutbox(key string) ([]string, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), shconstant.CtxTimeoutRedis)
	defer cancel()
	_, batchEvent, errLMPop := o.SharedRedis.LMPop(ctxTimeout, "RIGHT", 1000, key).Result()
	if errLMPop != nil {
		o.Logger.Error("failed to outbox security event: " + errLMPop.Error())
		return nil, errLMPop
	}
	return batchEvent, nil
}
