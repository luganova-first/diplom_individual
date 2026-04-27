package service

import (
	"github.com/luganova-first/diplom_individual/internal/model"
	"github.com/luganova-first/diplom_individual/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserDataService struct {
	User     *model.UserData
	Repo     repository.UserRepository
}

func NewUserDataService(user *model.User, repo repository.UserRepository) *UserDataService {
	return &UserDataService{
		User: &model.UserData{
			Login:    user.Login,
			Password: user.Password,
		},
		Repo: repo,
	}
}

func (s *UserDataService) hashPassword() error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(s.User.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.User.PassHash = string(bytes)
	return nil
}

func (s *UserDataService) CheckPassword() bool {
	err := bcrypt.CompareHashAndPassword([]byte(s.User.PassHash), []byte(s.User.Password))
	return err == nil
}

func (s *UserDataService) AddNewUser() error {
	if err := s.hashPassword(); err != nil {
		return err
	}
	return s.Repo.InsertNewUser(s.User.Login, s.User.PassHash)
}

func (s *UserDataService) GetUserData() error {
	userID, passHash, err := s.Repo.SelectUserData(s.User.Login)
	if err != nil {
		return err
	}
	s.User.UserID = userID
	s.User.PassHash = passHash
	return nil
}

func (s *UserDataService) CheckUser() (bool, error) {
	if err := s.GetUserData(); err != nil {
		return false, err
	}
	return s.User.UserID != 0 && s.User.PassHash != "", nil
}

func (s *UserDataService) GetUserOrders() ([]model.OrderItem, error) {
	return s.Repo.(repository.OrderRepository).SelectUserOrders(s.User.UserID)
}

func (s *UserDataService) GetUserWithdrawals() ([]model.WithdrawOutputItem, error) {
	return s.Repo.(repository.WithdrawRepository).SelectUserWithdrawals(s.User.UserID)
}
