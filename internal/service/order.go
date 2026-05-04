package service

import (
	"context"
	"github.com/luganova-first/diplom_individual/internal/model"
	"github.com/luganova-first/diplom_individual/internal/repository"
)

type OrderDataService struct {
	Order *model.Order
	Repo  repository.OrderRepository
}

func NewOrderDataService(order model.Order, repo repository.OrderRepository) *OrderDataService {
	return &OrderDataService{
		Order: &model.Order{
			UserID: order.UserID,
			Number: order.Number,
		},
		Repo: repo,
	}
}

func (s *OrderDataService) CreateOrder(ctx context.Context) error {
	return s.Repo.InsertNewOrder(ctx, s.Order.UserID, s.Order.Number)
}

func (s *OrderDataService) GetOrderData(ctx context.Context) error {
	orderData, err := s.Repo.SelectOrder(ctx, s.Order.Number)
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

func (s *OrderDataService) UpdateOrder(ctx context.Context) error {
	return s.Repo.UpdateOrderAccrual(ctx, s.Order.Number, s.Order.Status, s.Order.Accrual)
}

func (s *OrderDataService) GetOrdersForAccrual(ctx context.Context) ([]string, error) {
	return s.Repo.SelectOrdersForAccrual(ctx)
}
