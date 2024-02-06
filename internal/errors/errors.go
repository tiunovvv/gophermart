package errors

import "errors"

var (
	ErrLoginAlreadySaved     = errors.New("full URL already saved")
	ErrOrderSavedByThisUser  = errors.New("order was saved by this user")
	ErrOrderSavedByOtherUser = errors.New("order was saved by other user")
	ErrWithdrawAlreadySaved  = errors.New("withdraw URL already saved")
	ErrNoMoney               = errors.New("no money")
)
