package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/luganova-first/diplom_individual/internal/config"
	"github.com/luganova-first/diplom_individual/internal/model"
	"github.com/luganova-first/diplom_individual/internal/repository"
	"github.com/luganova-first/diplom_individual/internal/service"
	"github.com/luganova-first/diplom_individual/internal/userauth"
	"github.com/luganova-first/diplom_individual/pkg"
	"io"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	repo repository.Repository
	cfg  *config.Config
}

func NewHandler(repo repository.Repository, cfg *config.Config) *Handler {
	return &Handler{
		repo: repo,
		cfg:  cfg,
	}
}

// Хендлер регистрации пользователя
func (h *Handler) UserRegister() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "application/json" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		var user *model.User
		var buf bytes.Buffer

		_, err := buf.ReadFrom(req.Body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		err = json.Unmarshal(buf.Bytes(), &user)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		userService := service.NewUserDataService(user, h.repo)

		err = userService.AddNewUser()
		if err != nil {
			if errors.Is(err, repository.ErrLoginAlreadyExists) {
				res.WriteHeader(http.StatusConflict)
			} else {
				log.Println(err)
				res.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		err = userService.GetUserData()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		tokenString, err := userauth.BuildJWTString(user.Login)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		cookie := &http.Cookie{
			Name:     "jwt",
			Value:    tokenString,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		}

		http.SetCookie(res, cookie)
		res.WriteHeader(http.StatusOK)
	}
}

// Хендлер аутентификации пользователя
func (h *Handler) UserLogin() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "application/json" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		var user *model.User
		var buf bytes.Buffer

		_, err := buf.ReadFrom(req.Body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		err = json.Unmarshal(buf.Bytes(), &user)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		userService := service.NewUserDataService(user, h.repo)

		err = userService.GetUserData()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !userService.CheckPassword() {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		tokenString, err := userauth.BuildJWTString(user.Login)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		cookie := &http.Cookie{
			Name:     "jwt",
			Value:    tokenString,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		}

		http.SetCookie(res, cookie)
		res.WriteHeader(http.StatusOK)
	}
}

// Хендлер загрузки номера заказа
func (h *Handler) SetOrder() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "text/plain" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		userLogin := req.Context().Value("userLogin").(string)
		if userLogin == "" {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		user := &model.User{Login: userLogin}
		userService := service.NewUserDataService(user, h.repo)

		ok, err := userService.CheckUser()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		defer req.Body.Close()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(body) == 0 {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		number := string(body)

		if !pkg.ValidateLuhn(number) {
			res.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		order := model.Order{
			UserID: userService.User.UserID,
			Number: number,
		}
		orderService := service.NewOrderDataService(order, h.repo)

		err = orderService.GetOrderData()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if orderService.Order.OrderID > 0 && orderService.Order.Number == number && orderService.Order.UserID == userService.User.UserID {
			res.WriteHeader(http.StatusOK)
			return
		}

		if orderService.Order.OrderID > 0 && orderService.Order.Number == number && orderService.Order.UserID != userService.User.UserID {
			res.WriteHeader(http.StatusConflict)
			return
		}

		orderService.Order.UserID = userService.User.UserID
		err = orderService.CreateOrder()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusAccepted)
		h.GetOrdersAccrual()
	}
}

// Хендлер получения списка загруженных номеров заказов
func (h *Handler) GetOrders() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		userLogin := req.Context().Value("userLogin").(string)
		if userLogin == "" {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		user := &model.User{
			Login:    userLogin,
			Password: "",
		}

		userService := service.NewUserDataService(user, h.repo)

		ok, err := userService.CheckUser()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		orders, err := userService.GetUserOrders()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(orders) == 0 {
			res.WriteHeader(http.StatusNoContent)
			return
		}

		// Отправляем ответ
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		json.NewEncoder(res).Encode(orders)
	}
}

// Хендлер запроса на списание средств
func (h *Handler) SetWithdraw() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// POST запрос должен быть с Content-Type `application/json`
		if req.Header.Get("Content-Type") != "application/json" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		userLogin := req.Context().Value("userLogin").(string)
		if userLogin == "" {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		user := &model.User{
			Login:    userLogin,
			Password: "",
		}

		userService := service.NewUserDataService(user, h.repo)

		ok, err := userService.CheckUser()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		var jsonData *model.WithdrawInputItem
		var buf bytes.Buffer

		// читаем тело запроса
		_, err = buf.ReadFrom(req.Body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		// десериализуем JSON в url
		err = json.Unmarshal(buf.Bytes(), &jsonData)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		if !pkg.ValidateLuhn(jsonData.Order) {
			res.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		log.Println(jsonData)

		balanceService := service.NewBalanceService(userService.User, jsonData, h.repo)

		userCurrent, err := balanceService.GetCurrent()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		userWithdrawn, err := balanceService.GetWithdrawn()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		balance := userCurrent - userWithdrawn

		checkSum := float32(jsonData.Sum)
		if checkSum > balance {
			res.WriteHeader(http.StatusPaymentRequired)
			return
		}

		err = balanceService.CreateWithdraw()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusOK)
	}
}

// Хендлер получения списка запросов на списание средств
func (h *Handler) GetWithdrawals() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		userLogin := req.Context().Value("userLogin").(string)
		if userLogin == "" {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		user := &model.User{
			Login:    userLogin,
			Password: "",
		}

		userService := service.NewUserDataService(user, h.repo)

		ok, err := userService.CheckUser()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		withdrawals, err := userService.GetUserWithdrawals()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(withdrawals) == 0 {
			res.WriteHeader(http.StatusNoContent)
			return
		}

		// Отправляем ответ
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		json.NewEncoder(res).Encode(withdrawals)
	}
}

// Хендлер получения баланса пользователя
func (h *Handler) GetBalance() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		userLogin := req.Context().Value("userLogin").(string)
		if userLogin == "" {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		user := &model.User{
			Login:    userLogin,
			Password: "",
		}

		userService := service.NewUserDataService(user, h.repo)

		ok, err := userService.CheckUser()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		var jsonData *model.WithdrawInputItem

		balanceService := service.NewBalanceService(userService.User, jsonData, h.repo)

		userCurrent, err := balanceService.GetCurrent()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		userWithdrawn, err := balanceService.GetWithdrawn()
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Println("BBBBBB111111111")
		log.Println(userCurrent)
		log.Println(userWithdrawn)

		balance := model.Balance{
			Current:   userCurrent,
			Withdrawn: userWithdrawn,
		}

		resp, err := json.MarshalIndent(balance, "", " ")
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Отправляем ответ
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		res.Write(resp)
	}
}

// Хендлер информации о расчёте начислений баллов лояльности.
func (h *Handler) GetOrdersAccrual() {
	orders, err := h.repo.SelectOrdersForAccrual()
	if err != nil {
		log.Println(err)
		return
	}

	if len(orders) == 0 {
		return
	}

	for _, order := range orders {
		log.Printf("Проверяем заказ %s\n", order)
		url := fmt.Sprintf("%s/api/orders/%s", h.cfg.AccrualAddr, order)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Println(err)
			return
		}

		req.Header.Set("Content-Length", "0")

		client := http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()

		// Обработка различных кодов ответа
		switch resp.StatusCode {
		case http.StatusOK:
			var jsonData model.OrderAccrual
			var buf bytes.Buffer

			// читаем тело запроса
			_, err = buf.ReadFrom(resp.Body)
			if err != nil {
				log.Println(err)
				return
			}

			// десериализуем JSON в url
			err = json.Unmarshal(buf.Bytes(), &jsonData)
			if err != nil {
				log.Println(err)
				return
			}

			order := model.Order{
				Number: jsonData.Order,
			}
			orderService := service.NewOrderDataService(order, h.repo)
			orderService.Order.Status = jsonData.Status
			orderService.Order.Accrual = jsonData.Accrual

			err = orderService.UpdateOrder()
			if err != nil {
				log.Println(err)
				return
			}

		case http.StatusNoContent:
			log.Printf("Заказ %s не зарегистрирован в системе расчёта\n", order)

		case http.StatusTooManyRequests:
			// Обработка ограничения частоты запросов
			retryAfter := resp.Header.Get("Retry-After")
			log.Printf("Превышен лимит запросов для заказа %s. Retry-After: %s\n",
				order, retryAfter)

			// Можно прочитать тело ответа
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Сообщение: %s\n", string(body))

			// Пауза перед следующим запросом
			if retryAfter != "" {
				if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
					time.Sleep(seconds)
				}
			}

		case http.StatusInternalServerError:
			log.Printf("Внутренняя ошибка сервера при запросе заказа %s\n", order)
		}
	}
}
