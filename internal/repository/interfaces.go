package repository

import (
	"github.com/luganova-first/diplom_individual/internal/model"
)

type UserRepository interface {
	InsertNewUser(login string, passHash string) error
	SelectUserData(login string) (int64, string, error)
	CheckUserExists(login string) (bool, error)
}

type OrderRepository interface {
	InsertNewOrder(userID int64, number string) error
	SelectOrder(number string) (model.Order, error)
	SelectUserOrders(userID int64) ([]model.OrderItem, error)
	SelectOrdersForAccrual() ([]string, error)
	UpdateOrderAccrual(orderData model.OrderAccrual) error
}

type WithdrawRepository interface {
	InsertNewWithdraw(userID int64, order string, sum float32) error
	SelectUserWithdrawals(userID int64) ([]model.WithdrawOutputItem, error)
	SelectCurrent(userID int64) (float32, error)
	SelectWithdrawn(userID int64) (float32, error)
}

// Composite интерфейс для удобства
type Repository interface {
	UserRepository
	OrderRepository
	WithdrawRepository
	Close() error
}
