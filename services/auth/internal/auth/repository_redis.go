package auth

import (
	"app/auth-service/internal/apperrors"
	"app/auth-service/internal/common"
	"context"
	"errors"
	"shared/loggers"
	"shared/shconstant"
	"shared/storage"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type IRepositoryRedis interface {
	CreateUserSession(ctxRequest context.Context, sessionID, action string, dataUser map[string]any) error
	CreateRefresh(ctxRequest context.Context, params *CreateRefreshParams) error
	LogoutRefresh(ctxRequest context.Context, userUUID string, refreshKey string) error
	GetRefreshData(ctxRequest context.Context, userUUID, refreshUUID string) (*common.RefreshData, string, error)
	RotationRefresh(ctxRequest context.Context, userUUID, newRefreshKey, oldRefreshKey string) error
	GetUserSession(ctxRequest context.Context, sessionID, action string) (map[string]string, error)
	DeleteUserRefreshes(ctxRequest context.Context, userUUID string) error
}
type RepositoryRedisAuth struct {
	*storage.Redis
	*loggers.Logger
}

func NewRepositoryRedis(redis *storage.Redis, logger *loggers.Logger) IRepositoryRedis {
	return &RepositoryRedisAuth{
		Redis:  redis,
		Logger: logger,
	}
}

const (
	SessionKey       = "session:"
	UserRefreshesKey = "user_refreshes:"
	NullByte         = "\x00"
)

func (r *RepositoryRedisAuth) CreateUserSession(ctxRequest context.Context, sessionID, action string, dataUser map[string]any) error {
	ctxTimeout, cancel := context.WithTimeout(ctxRequest, shconstant.CtxTimeoutRedis)
	defer cancel()
	keySession := SessionKey + action + ":" + sessionID
	dataUser[common.AttemptsLeft] = 5
	pipe := r.Redis.Pipeline()
	pipe.HSet(ctxTimeout, keySession, dataUser)
	pipe.Expire(ctxTimeout, keySession, common.TTLSessionJWT)
	if _, errPipeline := pipe.Exec(ctxTimeout); errPipeline != nil {
		r.Logger.Error("failed to create user session: " + errPipeline.Error())
		return errPipeline
	}
	return nil
}
func (r *RepositoryRedisAuth) GetUserSession(ctxRequest context.Context, sessionID, action string) (map[string]string, error) {
	ctxTimeout, cancel := context.WithTimeout(ctxRequest, shconstant.CtxTimeoutRedis)
	defer cancel()
	keySession := SessionKey + action + ":" + sessionID
	existKey, errCheckExist := r.Redis.Exists(ctxTimeout, keySession).Result()
	if errCheckExist != nil {
		r.Logger.Error("failed to check session existence: " + errCheckExist.Error())
		return nil, errCheckExist
	}
	if existKey != 1 {
		return nil, apperrors.ErrSessionExpired
	}
	pipe := r.Redis.Pipeline()
	pipe.HIncrBy(ctxTimeout, keySession, common.AttemptsLeft, -1)
	HGetAllCmd := pipe.HGetAll(ctxTimeout, keySession)
	if _, errPipeline := pipe.Exec(ctxTimeout); errPipeline != nil {
		r.Logger.Error("failed to get user session: " + errPipeline.Error())
		return nil, errPipeline
	}
	dataUser, errHGetAll := HGetAllCmd.Result()
	if errHGetAll != nil {
		r.Logger.Error("failed to get user session: " + errHGetAll.Error())
	}
	if len(dataUser) == 0 {
		r.Logger.Warn("session value is empty")
		return nil, apperrors.ErrSessionExpired
	}
	attempts, errParse := strconv.Atoi(dataUser[common.AttemptsLeft])
	if errParse != nil {
		r.Logger.Error("failed to parse attempts left: " + errParse.Error())
		return nil, apperrors.ErrSessionExpired
	}
	if attempts < 0 {
		if errDel := r.Redis.Del(ctxTimeout, keySession).Err(); errDel != nil {
			r.Logger.Error("failed to delete session: " + errDel.Error())
		}
		return nil, apperrors.ErrSessionExpired
	}
	return dataUser, nil
}

type CreateRefreshParams struct {
	UserUUID    string
	RefreshUUID string
	UserAgent   string
	IPUser      string
	EmailUser   string
}

func (r *RepositoryRedisAuth) CreateRefresh(ctxRequest context.Context, params *CreateRefreshParams) error {
	ctxTimeout, cancel := context.WithTimeout(ctxRequest, shconstant.CtxTimeoutRedis)
	defer cancel()
	keyUserRefreshes := UserRefreshesKey + params.UserUUID
	clearBefore := time.Now().Add(-common.TTLRefreshKey).Unix()
	keyRefresh := params.RefreshUUID + NullByte + params.UserAgent + NullByte + params.IPUser + NullByte + params.EmailUser
	pipe := r.Redis.Pipeline()
	pipe.ZRemRangeByScore(ctxTimeout, keyUserRefreshes, "0", strconv.FormatInt(clearBefore, 10))
	pipe.ZAdd(ctxTimeout, keyUserRefreshes, redis.Z{
		Member: keyRefresh,
		Score:  float64(time.Now().Unix()),
	})
	pipe.ZRemRangeByRank(ctxTimeout, keyUserRefreshes, 0, -6)
	pipe.Expire(ctxTimeout, keyUserRefreshes, common.TTLRefreshKey)
	if _, errCreateRefresh := pipe.Exec(ctxTimeout); errCreateRefresh != nil {
		r.Logger.Error("failed to create refresh: " + errCreateRefresh.Error())
		return errCreateRefresh
	}
	return nil
}

func (r *RepositoryRedisAuth) LogoutRefresh(ctxRequest context.Context, userUUID string, refreshKey string) error {
	ctxTimeout, cancel := context.WithTimeout(ctxRequest, shconstant.CtxTimeoutRedis)
	defer cancel()
	keyUserRefreshes := UserRefreshesKey + userUUID
	if errZRem := r.Redis.ZRem(ctxTimeout, keyUserRefreshes, refreshKey).Err(); errZRem != nil {
		r.Logger.Warn("failed to delete refresh data: " + errZRem.Error())
		return errZRem
	}
	return nil
}

func (r *RepositoryRedisAuth) GetRefreshData(ctxRequest context.Context, userUUID, refreshUUID string) (*common.RefreshData, string, error) {
	ctxTimeout, cancel := context.WithTimeout(ctxRequest, shconstant.CtxTimeoutRedis)
	defer cancel()
	keyUserRefreshes := UserRefreshesKey + userUUID
	mask := refreshUUID + NullByte + `*`
	iter := r.Redis.ZScan(ctxTimeout, keyUserRefreshes, 0, mask, 10).Iterator()
	var refresh string
	for iter.Next(ctxTimeout) {
		refresh = iter.Val()
		break
	}
	if iter.Err() != nil {
		r.Logger.Error("failed to iterate user_refreshes: " + iter.Err().Error())
		return nil, "", iter.Err()
	}
	if refresh == "" {
		return nil, "", errors.New("not found data refresh")
	}
	partKey := strings.Split(refresh, NullByte)
	if len(partKey) != 4 {
		return nil, "", errors.New("not found data refresh")
	}
	return &common.RefreshData{
		UserAgent: partKey[1],
		IP:        partKey[2],
		Email:     partKey[3],
	}, refresh, nil
}

func (r *RepositoryRedisAuth) RotationRefresh(ctxRequest context.Context, userUUID, newRefreshKey, oldRefreshKey string) error {
	ctxTimeout, cancel := context.WithTimeout(ctxRequest, shconstant.CtxTimeoutRedis)
	defer cancel()
	keyUserRefreshes := UserRefreshesKey + userUUID
	clearBefore := time.Now().Add(-common.TTLRefreshKey).Unix()
	pipe := r.Redis.Pipeline()
	pipe.ZRemRangeByScore(ctxTimeout, keyUserRefreshes, "0", strconv.FormatInt(clearBefore, 10))
	pipe.ZRem(ctxTimeout, keyUserRefreshes, oldRefreshKey)
	pipe.ZAdd(ctxTimeout, keyUserRefreshes, redis.Z{
		Member: newRefreshKey,
		Score:  float64(time.Now().Unix()),
	})
	pipe.ZRemRangeByRank(ctxTimeout, keyUserRefreshes, 0, -6)
	pipe.Expire(ctxTimeout, keyUserRefreshes, common.TTLRefreshKey)
	if _, errRotation := pipe.Exec(ctxTimeout); errRotation != nil {
		r.Logger.Error("failed to rotation refresh: " + errRotation.Error())
		return errRotation
	}
	return nil
}
func (r *RepositoryRedisAuth) DeleteUserRefreshes(ctxRequest context.Context, userUUID string) error {
	ctxTimeout, cancel := context.WithTimeout(ctxRequest, shconstant.CtxTimeoutRedis)
	defer cancel()
	keyUserRefreshes := UserRefreshesKey + userUUID
	if errDel := r.Redis.Del(ctxTimeout, keyUserRefreshes).Err(); errDel != nil {
		r.Logger.Error("failed to delete user refreshes: " + errDel.Error())
		return errDel
	}
	return nil
}
