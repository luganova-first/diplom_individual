package service

import (
	"github.com/luganova-first/diplom_individual/internal/config"
	"github.com/luganova-first/diplom_individual/internal/model"
	"github.com/luganova-first/diplom_individual/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserDataService model.UserData
type OrderService model.Order
type WithdrawService model.Withdraw

// NewShortenerService создает новый экземпляр сервиса
func NewUserData(user model.User) *UserDataService {
	return &UserDataService{
		Login:    user.Login,
		Password: user.Password,
		PassHash: "",
		UserID:   0,
	}
}

// HashPassword создает хеш пароля
func (u *UserDataService) HashPassword() error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.PassHash = string(bytes)

	return nil
}

// CheckPassword проверяет пароль на соответствие хешу
func (u *UserDataService) CheckPassword() bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PassHash), []byte(u.Password))
	return err == nil
}

// AddNewUser добавляет нового юзера
func (u *UserDataService) AddNewUser(cfg *config.Config) error {
	err := u.HashPassword()
	if err != nil {
		return err
	}

	err = repository.InsertNewUser(cfg, u.Login, u.PassHash)
	if err != nil {
		return err
	}

	return nil
}

// GetUserData получает данные юзера
func (u *UserDataService) GetUserData(cfg *config.Config) error {
	userID, passwordHash, err := repository.SelectUserData(cfg, u.Login)
	if err != nil {
		return err
	}

	u.UserID = userID
	u.PassHash = passwordHash

	return nil
}

// CheckUser проверяет существование юзера по логину
func (u *UserDataService) CheckUser(cfg *config.Config) (bool, error) {
	err := u.GetUserData(cfg)
	if err != nil {
		return false, err
	}

	if u.UserID != 0 && u.PassHash != "" {
		return true, nil
	}

	return false, nil
}

// GetUserOrders получение списка загруженных номеров заказов
func (u *UserDataService) GetUserOrders(cfg *config.Config) ([]model.OrderItem, error) {
	orders, err := repository.SelectUserOrders(cfg, u.UserID)
	if err != nil {
		return orders, err
	}

	return orders, nil
}

// GetUserWithdrawals получение списка запросов на списание средств
func (u *UserDataService) GetUserWithdrawals(cfg *config.Config) ([]model.WithdrawOutputItem, error) {
	withdrawals, err := repository.SelectUserWithdrawals(cfg, u.UserID)
	if err != nil {
		return withdrawals, err
	}

	return withdrawals, nil
}

func (o *OrderService) CreateOrder(cfg *config.Config) error {
	err := repository.InsertNewOrder(cfg, o.UserID, o.Number)
	if err != nil {
		return err
	}

	return nil
}

func (o *OrderService) GetOrderData(cfg *config.Config) error {
	orderData, err := repository.SelectOrder(cfg, o.Number)
	if err != nil {
		return err
	}

	o.OrderID = orderData.OrderID
	o.UserID = orderData.UserID
	o.Number = orderData.Number
	o.Status = orderData.Status
	o.Accrual = orderData.Accrual
	o.UploadedAt = orderData.UploadedAt

	return nil
}

func (w *WithdrawService) CreateWithdraw(cfg *config.Config) error {
	err := repository.InsertNewWithdraw(cfg, w.UserID, w.Order, w.Sum)
	if err != nil {
		return err
	}

	return nil
}

func GetCurrent(cfg *config.Config, userID int64) (float32, error) {
	current, err := repository.SelectCurrent(cfg, userID)
	if err != nil {
		return current, err
	}

	return current, nil
}

func GetWithdrawn(cfg *config.Config, userID int64) (float32, error) {
	withdrawn, err := repository.SelectWithdrawn(cfg, userID)
	if err != nil {
		return withdrawn, err
	}

	return withdrawn, nil
}

func GetOrdersForAccrual(cfg *config.Config) ([]string, error) {
	orders, err := repository.SelectOrdersForAccrual(cfg)
	if err != nil {
		return orders, err
	}

	return orders, nil
}

func UpdateOrder(cfg *config.Config, orderData model.OrderAccrual) error {
	err := repository.UpdateOrderAccrual(cfg, orderData)
	if err != nil {
		return err
	}

	return nil
}
