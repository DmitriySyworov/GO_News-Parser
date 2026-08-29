package pagination

import (
	"shared/sherrors"
	"strconv"
)

const (
	MaxLimit      = 100
	DefaultLimit  = 10
	DefaultOffset = 0
)

func HelperPagination(limitStr, offsetStr string, mapError *sherrors.MapError) (int, int) {
	var limit, offset int
	var errLimit, errOffset error
	if limitStr != "" {
		limit, errLimit = strconv.Atoi(limitStr)
		if errLimit != nil {
			mapError.Map["limit"] = sherrors.ErrIncorrectLimit.Error()
		}
	} else {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		mapError.Map["limit"] = sherrors.ErrIncorrectLimit.Error()
	} else if limit <= 0 {
		mapError.Map["limit"] = sherrors.ErrIncorrectLimit.Error()
	}
	if offsetStr != "" {
		offset, errOffset = strconv.Atoi(offsetStr)
		if errOffset != nil {
			mapError.Map["offset"] = sherrors.ErrIncorrectOffset.Error()
		}
	} else {
		offset = DefaultOffset
	}
	if offset < 0 {
		mapError.Map["offset"] = sherrors.ErrIncorrectOffset.Error()
	}
	return limit, offset
}
