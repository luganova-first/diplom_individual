package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/luganova-first/diplom_individual/internal/config"
	"github.com/luganova-first/diplom_individual/internal/model"
	"github.com/pressly/goose/v3"
	"time"
)

var ErrLoginAlreadyExists = errors.New("login already exists")

func DB(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DBconnStr)
	if err != nil {
		return db, err
	}

	err = UpDBMigrations(db)
	if err != nil {
		return db, err
	}

	return db, nil
}

func UpDBMigrations(db *sql.DB) error {
	if err := goose.Up(db, "./migrations"); err != nil {
		return fmt.Errorf("failed up migrations: %w", err)
	}

	return nil
}

func InsertNewUser(cfg *config.Config, login string, passHash string) error {
	db, err := DB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO users (login, password_hash) VALUES ($1, $2)", login, passHash)
	if err != nil {
		// Проверяем, является ли ошибка нарушением уникальности
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			// Нарушение уникальности - такой логин уже существует
			return fmt.Errorf("%w: %s", ErrLoginAlreadyExists, login)
		}
		return err
	}

	return nil
}

func SelectUserData(cfg *config.Config, login string) (int64, string, error) {
	db, err := DB(cfg)
	if err != nil {
		return 0, "", err
	}
	defer db.Close()

	row := db.QueryRowContext(context.Background(), "SELECT * FROM users WHERE login = $1", login)

	var userID int64
	var userLogin string
	var passwordHash string
	err = row.Scan(&userID, &userLogin, &passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", nil
		} else {
			return 0, "", err
		}
	}

	return userID, passwordHash, nil
}

func InsertNewOrder(cfg *config.Config, userID int64, number string) error {
	db, err := DB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO orders (user_id, number) VALUES ($1, $2)", userID, number)
	if err != nil {
		return err
	}

	return nil
}

func SelectOrder(cfg *config.Config, number string) (model.Order, error) {
	var o model.Order

	db, err := DB(cfg)
	if err != nil {
		return o, err
	}
	defer db.Close()

	row := db.QueryRowContext(context.Background(), "SELECT * FROM orders WHERE number = $1", number)

	err = row.Scan(&o.OrderID, &o.UserID, &o.Number, &o.Status, &o.Accrual, &o.UploadedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return o, nil
		} else {
			return o, err
		}
	}

	return o, nil
}

func SelectUserOrders(cfg *config.Config, userID int64) ([]model.OrderItem, error) {
	var orders []model.OrderItem

	db, err := DB(cfg)
	if err != nil {
		return orders, err
	}
	defer db.Close()

	query := `
		SELECT number, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := db.QueryContext(context.Background(), query, userID)
	if err != nil {
		return orders, err
	}
	defer rows.Close()

	// пробегаем по всем записям
	for rows.Next() {
		var number string
		var status string
		var accrual float32
		var uploadedAt time.Time
		err = rows.Scan(&number, &status, &accrual, &uploadedAt)
		if err != nil {
			return orders, err
		}

		orderItem := model.OrderItem{
			Number:     number,
			Status:     status,
			Accrual:    accrual,
			UploadedAt: uploadedAt.Format("2006-01-02T15:04:05-07:00"),
		}

		orders = append(orders, orderItem)
	}

	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return orders, err
	}

	return orders, nil
}

func InsertNewWithdraw(cfg *config.Config, userID int64, order string, sum float32) error {
	db, err := DB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO withdrawals (user_id, number, sum) VALUES ($1, $2, $3)", userID, order, sum)
	if err != nil {
		return err
	}

	return nil
}

func SelectUserWithdrawals(cfg *config.Config, userID int64) ([]model.WithdrawOutputItem, error) {
	var withdrawals []model.WithdrawOutputItem

	db, err := DB(cfg)
	if err != nil {
		return withdrawals, err
	}
	defer db.Close()

	query := `
		SELECT number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC
	`

	rows, err := db.QueryContext(context.Background(), query, userID)
	if err != nil {
		return withdrawals, err
	}
	defer rows.Close()

	// пробегаем по всем записям
	for rows.Next() {
		var order string
		var sum float32
		var processedAt time.Time
		err = rows.Scan(&order, &sum, &processedAt)
		if err != nil {
			return withdrawals, err
		}

		withdrawItem := model.WithdrawOutputItem{
			Order:       order,
			Sum:         sum,
			ProcessedAt: processedAt.Format("2006-01-02T15:04:05-07:00"),
		}

		withdrawals = append(withdrawals, withdrawItem)
	}

	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return withdrawals, err
	}

	return withdrawals, nil
}

func SelectCurrent(cfg *config.Config, userID int64) (float32, error) {
	db, err := DB(cfg)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var sum float32

	query := `
		SELECT COALESCE(SUM(accrual), 0)
		FROM orders
		WHERE user_id = $1
		AND status = 'PROCESSED'
	`

	err = db.QueryRowContext(context.Background(), query, userID).Scan(&sum)
	if err != nil {
		return 0, err
	}

	return sum, nil
}

func SelectWithdrawn(cfg *config.Config, userID int64) (float32, error) {
	db, err := DB(cfg)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var sum float32

	query := `
		SELECT COALESCE(SUM(sum), 0)
		FROM withdrawals
		WHERE user_id = $1
	`

	err = db.QueryRowContext(context.Background(), query, userID).Scan(&sum)
	if err != nil {
		return 0, err
	}

	return sum, nil
}

func SelectOrdersForAccrual(cfg *config.Config) ([]string, error) {
	var orders []string

	db, err := DB(cfg)
	if err != nil {
		return orders, err
	}
	defer db.Close()

	query := `
		SELECT number
		FROM orders
		WHERE status IN ('NEW', 'PROCESSING')
	`

	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		return orders, err
	}
	defer rows.Close()

	// пробегаем по всем записям
	for rows.Next() {
		var order string
		err = rows.Scan(&order)
		if err != nil {
			return orders, err
		}

		orders = append(orders, order)
	}

	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return orders, err
	}

	return orders, nil
}

func UpdateOrderAccrual(cfg *config.Config, orderData model.OrderAccrual) error {
	db, err := DB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	query := `
		UPDATE orders
		SET status = $1, accrual = $2
		WHERE number = $3
	`

	result, err := db.Exec(query, orderData.Status, orderData.Accrual, orderData.Order)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return errors.New("Order not found")
	}

	return nil
}
