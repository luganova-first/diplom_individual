package repository

import (
	"context"
	"github.com/luganova-first/diplom_individual/internal/model"
)

type UserRepository interface {
	InsertNewUser(ctx context.Context, login string, passHash string) error
	SelectUserData(ctx context.Context, login string) (int64, string, error)
	CheckUserExists(ctx context.Context, login string) (bool, error)
}

type OrderRepository interface {
	InsertNewOrder(ctx context.Context, userID int64, number string) error
	SelectOrder(ctx context.Context, number string) (model.Order, error)
	SelectUserOrders(ctx context.Context, userID int64) ([]model.OrderItem, error)
	SelectOrdersForAccrual(ctx context.Context) ([]string, error)
	UpdateOrderAccrual(ctx context.Context, number string, status string, accrual float32) error
}

type WithdrawRepository interface {
	InsertNewWithdraw(ctx context.Context, userID int64, order string, sum float32) error
	SelectUserWithdrawals(ctx context.Context, userID int64) ([]model.WithdrawOutputItem, error)
	SelectCurrent(ctx context.Context, userID int64) (float32, error)
	SelectWithdrawn(ctx context.Context, userID int64) (float32, error)
}

// Composite интерфейс для удобства
type Repository interface {
	UserRepository
	OrderRepository
	WithdrawRepository
	Close() error
}
