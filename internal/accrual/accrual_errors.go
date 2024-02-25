package accrual

import "errors"

var (
	errOrderNotRegistered = errors.New("order not registred")
	errTooManyRequests    = errors.New("too many requests")
	errAccrualServerError = errors.New("internal accrual server error")
)
