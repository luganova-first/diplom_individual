package service

import (
	"github.com/luganova-first/diplom_individual/internal/model"
	"github.com/luganova-first/diplom_individual/internal/repository"
)

type OrderDataService struct {
	Order *model.Order
	Repo  repository.OrderRepository
}

func NewOrderDataService(order model.Order, repo repository.OrderRepository) *OrderDataService {
	return &OrderDataService{
		Order: &order,
		Repo:  repo,
	}
}

func (s *OrderDataService) CreateOrder() error {
	return s.Repo.InsertNewOrder(s.Order.UserID, s.Order.Number)
}

func (s *OrderDataService) GetOrderData() error {
	orderData, err := s.Repo.SelectOrder(s.Order.Number)
	if err != nil {
		return err
	}
	s.Order.OrderID = orderData.OrderID
	s.Order.UserID = orderData.UserID
	s.Order.Status = orderData.Status
	s.Order.Accrual = orderData.Accrual
	s.Order.UploadedAt = orderData.UploadedAt
	return nil
}

func (s *OrderDataService) UpdateOrder(orderData model.OrderAccrual) error {
	return s.Repo.UpdateOrderAccrual(orderData)
}

func (s *OrderDataService) GetOrdersForAccrual() ([]string, error) {
	return s.Repo.SelectOrdersForAccrual()
}
