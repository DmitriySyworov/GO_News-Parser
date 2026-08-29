package shconstant

import "time"

const (
	TypeSoftDelete           = "soft-delete"
	TypeHardDelete           = "hard-delete"
	CtxTimeoutRedis          = time.Millisecond * 250
	CtxTimeoutSendEventKafka = time.Second * 15
	EventKafkaTTL            = time.Hour * 24
	EventKey                 = "event:"

	ServiceName = "Budget Planner"
	Domain      = "budget-planner"
)
