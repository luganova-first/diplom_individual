package main

import (
	"github.com/luganova-first/diplom_individual/internal/archiver"
	"github.com/luganova-first/diplom_individual/internal/config"
	"github.com/luganova-first/diplom_individual/internal/handler"
	"github.com/luganova-first/diplom_individual/internal/logger"
	"github.com/luganova-first/diplom_individual/internal/userauth"

	"log"
	"net/http"
	// "time"

	"github.com/go-chi/chi/v5"
)

func main() {
	// Инициализация конфигурации
	cfg := config.NewConfig()

	// Валидация конфигурации
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	//ticker := time.NewTicker(2 * time.Second)
	//defer ticker.Stop()

	// Запускаем горутину запросов accrual
	//go func() {
	//	for range ticker.C {
	//		handler.GetOrdersAccrual(cfg)
	//	}
	//}()

	r := chi.NewRouter()

	// Передаем базовый URL в хендлер
	r.Post("/api/user/register", handler.UserRegister(cfg))
	r.Post("/api/user/login", handler.UserLogin(cfg))
	r.Post("/api/user/orders", handler.SetOrder(cfg))
	r.Post("/api/user/balance/withdraw", handler.SetWithdraw(cfg))
	r.Get("/api/user/orders", handler.GetOrders(cfg))
	r.Get("/api/user/withdrawals", handler.GetWithdrawals(cfg))
	r.Get("/api/user/balance", handler.GetBalance(cfg))

	r.MethodNotAllowed(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusBadRequest)
	})

	log.Fatal(http.ListenAndServe(cfg.ServerAddress, archiver.GzipHandler(logger.WithLogging(userauth.GetUserCookie(r)))))
}
