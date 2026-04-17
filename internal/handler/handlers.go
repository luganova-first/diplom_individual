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

// Хендлер регистрации пользователя
func UserRegister(cfg *config.Config) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// POST запрос должен быть с Content-Type `application/json`
		if req.Header.Get("Content-Type") != "application/json" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		var user model.User
		var buf bytes.Buffer

		// читаем тело запроса
		_, err := buf.ReadFrom(req.Body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		// десериализуем JSON в user
		err = json.Unmarshal(buf.Bytes(), &user)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		u := service.NewUserData(user)

		err = u.AddNewUser(cfg)
		if err != nil {
			if errors.Is(err, repository.ErrLoginAlreadyExists) {
				res.WriteHeader(http.StatusConflict)
			} else {
				log.Println(err)
				res.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			err = u.GetUserData(cfg)
			if err != nil {
				log.Println(err)
				res.WriteHeader(http.StatusInternalServerError)
			}

			tokenString, err := userauth.BuildJWTString(u.Login)
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
}

// Хендлер аутентификации пользователя
func UserLogin(cfg *config.Config) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// POST запрос должен быть с Content-Type `application/json`
		if req.Header.Get("Content-Type") != "application/json" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		var user model.User
		var buf bytes.Buffer

		// читаем тело запроса
		_, err := buf.ReadFrom(req.Body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		// десериализуем JSON в user
		err = json.Unmarshal(buf.Bytes(), &user)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		u := service.NewUserData(user)

		err = u.GetUserData(cfg)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if u.PassHash == "" || !u.CheckPassword() {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		tokenString, err := userauth.BuildJWTString(u.Login)
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
func SetOrder(cfg *config.Config) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		// POST запрос должен быть с Content-Type `text/plain`
		if req.Header.Get("Content-Type") != "text/plain" {
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

		u := service.NewUserData(*user)
		ok, err := u.CheckUser(cfg)
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

		if string(body) == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		number := string(body)

		if !pkg.ValidateLuhn(number) {
			res.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		order := &service.OrderService{
			UserID: u.UserID,
			Number: number,
		}

		err = order.GetOrderData(cfg)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if order.Number == number && order.UserID == u.UserID {
			res.WriteHeader(http.StatusOK)
			return
		}

		if order.Number == number && order.UserID != u.UserID {
			res.WriteHeader(http.StatusConflict)
			return
		}

		order.UserID = u.UserID
		order.Number = number

		err = order.CreateOrder(cfg)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusAccepted)

		// Попробуем сразу же получить accrual по заказу
		GetOrdersAccrual(cfg)
	}
}

// Хендлер получения списка загруженных номеров заказов
func GetOrders(cfg *config.Config) http.HandlerFunc {
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

		u := service.NewUserData(*user)
		ok, err := u.CheckUser(cfg)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		orders, err := u.GetUserOrders(cfg)
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

		// Проверяем accrual по всем заказам
		GetOrdersAccrual(cfg)
	}
}

// Хендлер запроса на списание средств
func SetWithdraw(cfg *config.Config) http.HandlerFunc {
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

		u := service.NewUserData(*user)
		ok, err := u.CheckUser(cfg)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		var jsonData model.WithdrawInputItem
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

		userCurrent, err := service.GetCurrent(cfg, u.UserID)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		userWithdrawn, err := service.GetWithdrawn(cfg, u.UserID)
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

		withdraw := &service.WithdrawService{
			UserID: u.UserID,
			Order:  jsonData.Order,
			Sum:    jsonData.Sum,
		}

		log.Println("11111111111111111")
		log.Println(jsonData)

		err = withdraw.CreateWithdraw(cfg)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusOK)

		// Проверяем accrual по всем заказам
		GetOrdersAccrual(cfg)
	}
}

// Хендлер получения списка запросов на списание средств
func GetWithdrawals(cfg *config.Config) http.HandlerFunc {
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

		u := service.NewUserData(*user)
		ok, err := u.CheckUser(cfg)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Проверяем accrual по всем заказам
		GetOrdersAccrual(cfg)

		withdrawals, err := u.GetUserWithdrawals(cfg)
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
func GetBalance(cfg *config.Config) http.HandlerFunc {
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

		u := service.NewUserData(*user)
		ok, err := u.CheckUser(cfg)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Перед балансом ещё раз проверяем Accrual по всем заказам
		GetOrdersAccrual(cfg)

		userCurrent, err := service.GetCurrent(cfg, u.UserID)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		userWithdrawn, err := service.GetWithdrawn(cfg, u.UserID)
		if err != nil {
			log.Println(err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		userCurrent = userCurrent - userWithdrawn

		log.Println("Current")
		log.Println(userCurrent)

		log.Println("Withdrawn")
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
func GetOrdersAccrual(cfg *config.Config) {
	// Получение списка заказов требующих информации о расчёте начислений баллов лояльности.
	orders, err := service.GetOrdersForAccrual(cfg)
	if err != nil {
		log.Println(err)
		return
	}

	if len(orders) == 0 {
		return
	}

	for _, order := range orders {
		log.Printf("Проверяем заказ %s\n", order)
		url := fmt.Sprintf("%s/api/orders/%s", cfg.AccrualAddr, order)

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

			log.Println("22222222222222222222")
			log.Println(jsonData)

			err = service.UpdateOrder(cfg, jsonData)
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
