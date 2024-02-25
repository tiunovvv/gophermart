package models

import "time"

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Order struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

type OrderWithTime struct {
	UploadedAt time.Time `json:"uploaded_at"`
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type Withdraw struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type Withdrawals struct {
	ProcessedAt time.Time `json:"processed_at"`
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
}
