package service

import (
	"github.com/luganova-first/diplom_individual/internal/model"
	"github.com/luganova-first/diplom_individual/internal/repository"
)

type BalanceService struct {
	User     *model.UserData
	Withdraw *model.WithdrawInputItem
	WithdrawRepo repository.WithdrawRepository
}

func NewBalanceService(user *model.User, withdraw *model.WithdrawInputItem, repo repository.Repository) *BalanceService {
	return &BalanceService{
		User: &model.UserData{
			Login:    user.Login,
			Password: user.Password,
		},
		Withdraw: &model.WithdrawInputItem{
			Order:    withdraw.Order,
			Sum: withdraw.Sum,
		},
		WithdrawRepo: repo,
	}
}

func (s *BalanceService) GetCurrent() (float32, error) {
	return s.WithdrawRepo.SelectCurrent(s.User.UserID)
}

func (s *BalanceService) GetWithdrawn() (float32, error) {
	return s.WithdrawRepo.SelectWithdrawn(s.User.UserID)
}

func (s *BalanceService) CreateWithdraw() error {
	return s.WithdrawRepo.InsertNewWithdraw(s.User.UserID, s.Withdraw.Order, s.Withdraw.Sum)
}
