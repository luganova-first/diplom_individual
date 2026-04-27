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

type PostgresRepository struct {
	db *sql.DB
	cfg *config.Config
}

func NewPostgresRepository(cfg *config.Config) (*PostgresRepository, error) {
	db, err := sql.Open("pgx", cfg.DBconnStr)
	if err != nil {
		return nil, err
	}

	if err := UpDBMigrations(db); err != nil {
		return nil, err
	}

	return &PostgresRepository{
		db: db,
		cfg: cfg,
	}, nil
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

func (r *PostgresRepository) InsertNewUser(login string, passHash string) error {
	_, err := r.db.Exec("INSERT INTO users (login, password_hash) VALUES ($1, $2)", login, passHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("%w: %s", ErrLoginAlreadyExists, login)
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) SelectUserData(login string) (int64, string, error) {
	row := r.db.QueryRowContext(context.Background(), "SELECT * FROM users WHERE login = $1", login)

	var userID int64
	var userLogin string
	var passwordHash string
	err := row.Scan(&userID, &userLogin, &passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", nil
		}
		return 0, "", err
	}

	return userID, passwordHash, nil
}

func (r *PostgresRepository) CheckUserExists(login string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE login = $1)"
	err := r.db.QueryRowContext(context.Background(), query, login).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) InsertNewOrder(userID int64, number string) error {
	_, err := r.db.Exec("INSERT INTO orders (user_id, number) VALUES ($1, $2)", userID, number)
	return err
}

func (r *PostgresRepository) SelectOrder(number string) (model.Order, error) {
	var o model.Order
	row := r.db.QueryRowContext(context.Background(), "SELECT * FROM orders WHERE number = $1", number)

	err := row.Scan(&o.OrderID, &o.UserID, &o.Number, &o.Status, &o.Accrual, &o.UploadedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return o, nil
		}
		return o, err
	}
	return o, nil
}

func (r *PostgresRepository) SelectUserOrders(userID int64) ([]model.OrderItem, error) {
	var orders []model.OrderItem

	query := `
		SELECT number, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := r.db.QueryContext(context.Background(), query, userID)
	if err != nil {
		return orders, err
	}
	defer rows.Close()

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

	return orders, rows.Err()
}

func (r *PostgresRepository) InsertNewWithdraw(userID int64, order string, sum float32) error {
	_, err := r.db.Exec("INSERT INTO withdrawals (user_id, number, sum) VALUES ($1, $2, $3)", userID, order, sum)
	return err
}

func (r *PostgresRepository) SelectUserWithdrawals(userID int64) ([]model.WithdrawOutputItem, error) {
	var withdrawals []model.WithdrawOutputItem

	query := `
		SELECT number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC
	`

	rows, err := r.db.QueryContext(context.Background(), query, userID)
	if err != nil {
		return withdrawals, err
	}
	defer rows.Close()

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

	return withdrawals, rows.Err()
}

func (r *PostgresRepository) SelectCurrent(userID int64) (float32, error) {
	var sum float32
	query := `
		SELECT COALESCE(SUM(accrual), 0)
		FROM orders
		WHERE user_id = $1
		AND status = 'PROCESSED'
	`
	err := r.db.QueryRowContext(context.Background(), query, userID).Scan(&sum)
	return sum, err
}

func (r *PostgresRepository) SelectWithdrawn(userID int64) (float32, error) {
	var sum float32
	query := `
		SELECT COALESCE(SUM(sum), 0)
		FROM withdrawals
		WHERE user_id = $1
	`
	err := r.db.QueryRowContext(context.Background(), query, userID).Scan(&sum)
	return sum, err
}

func (r *PostgresRepository) SelectOrdersForAccrual() ([]string, error) {
	var orders []string
	query := `
		SELECT number
		FROM orders
		WHERE status IN ('NEW', 'PROCESSING')
	`
	rows, err := r.db.QueryContext(context.Background(), query)
	if err != nil {
		return orders, err
	}
	defer rows.Close()

	for rows.Next() {
		var order string
		err = rows.Scan(&order)
		if err != nil {
			return orders, err
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

func (r *PostgresRepository) UpdateOrderAccrual(orderData model.OrderAccrual) error {
	query := `
		UPDATE orders
		SET status = $1, accrual = $2
		WHERE number = $3
	`
	result, err := r.db.Exec(query, orderData.Status, orderData.Accrual, orderData.Order)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return errors.New("order not found")
	}
	return nil
}

func UpDBMigrations(db *sql.DB) error {
	if err := goose.Up(db, "./migrations"); err != nil {
		return fmt.Errorf("failed up migrations: %w", err)
	}
	return nil
}