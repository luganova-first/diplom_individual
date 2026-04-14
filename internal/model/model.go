package model

import (
	"time"
)

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type UserData struct {
	Login    string
	Password string
	PassHash string
	UserID   int64
}

type Order struct {
	OrderID    int64
	UserID     int64
	Number     string
	Status     string
	Accrual    int
	UploadedAt time.Time
}

type OrderItem struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	Accrual    int    `json:"accrual"`
	UploadedAt string `json:"uploaded_at"`
}

type Withdraw struct {
	OrderID     int64
	UserID      int64
	Order       string
	Sum         int
	ProcessedAt time.Time
}

type WithdrawInputItem struct {
	Order string `json:"order"`
	Sum   int    `json:"sum"`
}

type WithdrawOutputItem struct {
	Order       string `json:"order"`
	Sum         int    `json:"sum"`
	ProcessedAt string `json:"processed_at"`
}

type Balance struct {
	Current   int64 `json:"current"`
	Withdrawn int64 `json:"withdrawn"`
}

type OrderAccrual struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}
