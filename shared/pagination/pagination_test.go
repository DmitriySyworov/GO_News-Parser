package pagination_test

import (
	"shared/pagination"
	"shared/sherrors"
	"testing"
)

var CasePaginationDataSuccess = []struct {
	TestName       string
	LimitStr       string
	OffsetStr      string
	LimitExpected  int
	OffsetExpected int
}{
	{TestName: "limit empty offset empty", LimitStr: "", OffsetStr: "", LimitExpected: pagination.DefaultLimit, OffsetExpected: pagination.DefaultOffset},
	{TestName: "limit 100 offset 0", LimitStr: "100", OffsetStr: "0", LimitExpected: 100, OffsetExpected: 0},
	{TestName: "limit empty offset 1000", LimitStr: "", OffsetStr: "1000", LimitExpected: pagination.DefaultLimit, OffsetExpected: 1000},
	{TestName: "limit 10 offset empty", LimitStr: "10", OffsetStr: "", LimitExpected: 10, OffsetExpected: pagination.DefaultOffset},
	{TestName: "limit is exactly max limit (100)", LimitStr: "100", OffsetStr: "10", LimitExpected: 100, OffsetExpected: 10},
	{TestName: "limit is exactly minimum valid (1)", LimitStr: "1", OffsetStr: "5", LimitExpected: 1, OffsetExpected: 5},
	{TestName: "offset is big but valid", LimitStr: "20", OffsetStr: "999999", LimitExpected: 20, OffsetExpected: 999999},
}

func TestHelperPaginationSuccess(t *testing.T) {
	for _, test := range CasePaginationDataSuccess {
		t.Run(test.TestName, func(t *testing.T) {
			mapError := &sherrors.MapError{Map: make(map[string]string, 2)}
			limit, offset := pagination.HelperPagination(test.LimitStr, test.OffsetStr, mapError)
			if len(mapError.Map) != 0 {
				t.Fatalf("unexpected errors: limit error: %s, offset error: %s", mapError.Map["limit"], mapError.Map["offset"])
			}
			if limit != test.LimitExpected {
				t.Errorf("limit: expected %d, got %d", test.LimitExpected, limit)
			}
			if offset != test.OffsetExpected {
				t.Errorf("offset: expected %d, got %d", test.OffsetExpected, offset)
			}
		})
	}
}

var CasePaginationDataNegative = []struct {
	TestName               string
	LimitStr               string
	OffsetStr              string
	ExpectedQuantityErrors int
	CheckLimitError        bool
	CheckOffsetError       bool
}{
	{TestName: "limit asadsd offset asdd", LimitStr: "asadsd", OffsetStr: "asdd", ExpectedQuantityErrors: 2, CheckLimitError: true, CheckOffsetError: true},
	{TestName: "limit 101 offset 0", LimitStr: "101", OffsetStr: "0", ExpectedQuantityErrors: 1, CheckLimitError: true, CheckOffsetError: false},
	{TestName: "limit 0 offset 1000", LimitStr: "0", OffsetStr: "1000", ExpectedQuantityErrors: 1, CheckLimitError: true, CheckOffsetError: false},
	{TestName: "limit -1 offset 1222", LimitStr: "-1", OffsetStr: "1222", ExpectedQuantityErrors: 1, CheckLimitError: true, CheckOffsetError: false},
	{TestName: "limit 2 offset -1", LimitStr: "2", OffsetStr: "-1", ExpectedQuantityErrors: 1, CheckLimitError: false, CheckOffsetError: true},
	{TestName: "limit with spaces", LimitStr: " 10 ", OffsetStr: "0", ExpectedQuantityErrors: 1, CheckLimitError: true, CheckOffsetError: false}, // strconv.Atoi не тримит пробелы
	{TestName: "offset with spaces", LimitStr: "10", OffsetStr: " 5", ExpectedQuantityErrors: 1, CheckLimitError: false, CheckOffsetError: true},
	{TestName: "limit overflow int", LimitStr: "999999999999999999999999999999", OffsetStr: "0", ExpectedQuantityErrors: 1, CheckLimitError: true, CheckOffsetError: false},
	{TestName: "offset overflow int", LimitStr: "10", OffsetStr: "999999999999999999999999999999", ExpectedQuantityErrors: 1, CheckLimitError: false, CheckOffsetError: true},
	{TestName: "limit float string", LimitStr: "10.5", OffsetStr: "0", ExpectedQuantityErrors: 1, CheckLimitError: true, CheckOffsetError: false},
}

func TestHelperPaginationNegative(t *testing.T) {
	for _, test := range CasePaginationDataNegative {
		t.Run(test.TestName, func(t *testing.T) {
			mapError := &sherrors.MapError{Map: make(map[string]string, 2)}
			pagination.HelperPagination(test.LimitStr, test.OffsetStr, mapError)

			if len(mapError.Map) == 0 {
				t.Fatal("expected errors, but got none")
			}
			if len(mapError.Map) != test.ExpectedQuantityErrors {
				t.Errorf("errors count: expected %d, got %d", test.ExpectedQuantityErrors, len(mapError.Map))
			}
			if test.CheckLimitError {
				if _, exists := mapError.Map["limit"]; !exists {
					t.Error("expected error for 'limit' field, but it's missing")
				}
			}
			if test.CheckOffsetError {
				if _, exists := mapError.Map["offset"]; !exists {
					t.Error("expected error for 'offset' field, but it's missing")
				}
			}
		})
	}
}
