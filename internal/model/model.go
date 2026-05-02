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
	Accrual    float32
	UploadedAt time.Time
}

type OrderItem struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float32 `json:"accrual"`
	UploadedAt string  `json:"uploaded_at"`
}

type Withdraw struct {
	OrderID     int64
	UserID      int64
	Order       string
	Sum         float32
	ProcessedAt time.Time
}

type WithdrawInputItem struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

type WithdrawOutputItem struct {
	Order       string  `json:"order"`
	Sum         float32 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

type Balance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

type OrderAccrual struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual"`
}
