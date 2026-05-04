package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/luganova-first/diplom_individual/internal/archiver"
	"github.com/luganova-first/diplom_individual/internal/config"
	"github.com/luganova-first/diplom_individual/internal/handler"
	"github.com/luganova-first/diplom_individual/internal/logger"
	"github.com/luganova-first/diplom_individual/internal/repository"
	"github.com/luganova-first/diplom_individual/internal/userauth"

	"github.com/go-chi/chi/v5"
)

func main() {
	// Инициализация конфигурации
	cfg := config.NewConfig()

	// Валидация конфигурации
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	// Инициализируем репозиторий
	repo, err := repository.NewPostgresRepository(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()

	// Создаем хендлер с зависимостями
	handler := handler.NewHandler(repo, cfg)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Контекст для graceful shutdown горутины с ticker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запускаем горутину запросов accrual
	go func() {
		for {
			select {
			case <-ticker.C:
				handler.GetOrdersAccrual()
			case <-ctx.Done():
				log.Println("Accrual worker stopped gracefully")
				return
			}
		}
	}()

	r := chi.NewRouter()

	// Передаем базовый URL в хендлер
	r.Post("/api/user/register", handler.UserRegister())
	r.Post("/api/user/login", handler.UserLogin())
	r.Post("/api/user/orders", handler.SetOrder())
	r.Post("/api/user/balance/withdraw", handler.SetWithdraw())
	r.Get("/api/user/orders", handler.GetOrders())
	r.Get("/api/user/withdrawals", handler.GetWithdrawals())
	r.Get("/api/user/balance", handler.GetBalance())

	r.MethodNotAllowed(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusBadRequest)
	})

	// Создаем HTTP сервер
	srv := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      archiver.GzipHandler(logger.WithLogging(userauth.GetUserCookie(r))),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("Starting server on %s", cfg.ServerAddress)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Ожидание сигнала остановки
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down gracefully...")

	// Даем время на завершение текущих запросов
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Останавливаем HTTP сервер
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Останавливаем горутину с ticker
	cancel()

	// Даем время горутине на завершение
	time.Sleep(1 * time.Second)

	log.Println("Server stopped gracefully")
}
