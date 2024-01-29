package errors

import "errors"

var (
	ErrLoginAlreadySaved    = errors.New("full URL already saved")
	ErrWithdrawAlreadySaved = errors.New("withdraw URL already saved")
)
