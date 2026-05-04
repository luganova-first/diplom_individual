package service

import (
	"context"
	"github.com/luganova-first/diplom_individual/internal/model"
	"github.com/luganova-first/diplom_individual/internal/repository"
)

type BalanceService struct {
	User         *model.UserData
	Withdraw     *model.WithdrawInputItem
	WithdrawRepo repository.WithdrawRepository
}

func NewBalanceService(user *model.UserData, withdraw *model.WithdrawInputItem, repo repository.Repository) *BalanceService {
	return &BalanceService{
		User: &model.UserData{
			Login:  user.Login,
			UserID: user.UserID,
		},
		Withdraw:     withdraw,
		WithdrawRepo: repo,
	}
}

func (s *BalanceService) GetCurrent() (float32, error) {
	return s.WithdrawRepo.SelectCurrent(context.Background(), s.User.UserID)
}

func (s *BalanceService) GetWithdrawn() (float32, error) {
	return s.WithdrawRepo.SelectWithdrawn(context.Background(), s.User.UserID)
}

func (s *BalanceService) CreateWithdraw() error {
	return s.WithdrawRepo.InsertNewWithdraw(context.Background(), s.User.UserID, s.Withdraw.Order, s.Withdraw.Sum)
}
