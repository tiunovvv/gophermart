package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	myErrors "github.com/tiunovvv/gophermart/internal/errors"
	"github.com/tiunovvv/gophermart/internal/models"
)

type DB struct {
	pool *pgxpool.Pool
	log  *zap.SugaredLogger
}

func NewDB(ctx context.Context, databaseURI string, log *zap.SugaredLogger) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(databaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the DSN: %w", err)
	}

	queryTracer := newQueryTracer(log)
	poolCfg.ConnConfig.Tracer = queryTracer
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	if err := runMigrations(databaseURI); err != nil {
		return nil, fmt.Errorf("failed to run DB migrations: %w", err)
	}

	database := &DB{pool: pool, log: log}
	return database, nil
}

//go:embed migrations/*.sql
var migrationsDir embed.FS

func runMigrations(databaseURI string) error {
	d, err := iofs.New(migrationsDir, "migrations")
	if err != nil {
		return fmt.Errorf("failed to return an iofs driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, databaseURI)
	if err != nil {
		return fmt.Errorf("failed to get a new migrate instance: %w", err)
	}
	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("failed to apply migrations to the DB: %w", err)
		}
	}
	return nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) NewUser(ctx context.Context, userID string, login string, hash string) error {
	const insertUser = `INSERT INTO users (user_id, login, pswd_hash) VALUES ($1, $2, $3) RETURNING login`
	var loginDB string
	err := db.pool.QueryRow(ctx, insertUser, userID, login, hash).Scan(&loginDB)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return myErrors.ErrLoginAlreadySaved
		}
		return fmt.Errorf("failed to insert new user: %w", err)
	}
	return nil
}

func (db *DB) GetUserID(ctx context.Context, login string) (string, string, error) {
	const selectUserID = `SELECT user_id, pswd_hash FROM users WHERE login = $1;`
	row := db.pool.QueryRow(ctx, selectUserID, login)
	var userID, hash string
	if err := row.Scan(&userID, &hash); err != nil {
		return "", "", fmt.Errorf("failed to get data from users: %w", err)
	}
	return userID, hash, nil
}

func (db *DB) SaveOrder(ctx context.Context, userID string, number string) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			db.log.Infof("failed to rollback: %w", err)
		}
	}()

	var userIDDB, numberDB string
	const selectOrder = `SELECT user_id, number FROM users_orders WHERE number = $1;`
	if err := tx.QueryRow(ctx, selectOrder, number).Scan(&userIDDB, &numberDB); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to check order in db: %w", err)
		}
	}

	if numberDB == number && userIDDB == userID {
		return myErrors.ErrOrderSavedByThisUser
	}

	if numberDB == number && userIDDB != userID {
		return myErrors.ErrOrderSavedByOtherUser
	}

	currentTime := time.Now()
	rfc3339String := currentTime.Format(time.RFC3339)
	const insertOrder = `
	INSERT INTO users_orders (number, status, user_id, uploaded_at) VALUES ($1, $2, $3, $4) RETURNING number`
	err = tx.QueryRow(ctx, insertOrder, number, `NEW`, userID, rfc3339String).Scan(&numberDB)
	if err != nil {
		return fmt.Errorf("failed to insert new order: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (db *DB) GetNewOrders(ctx context.Context) ([]models.OrderWithTime, error) {
	const select100NewOrders = `
	SELECT number, status, accrual, uploaded_at 
 	FROM users_orders WHERE status = 'NEW' OR status = 'PROCESSING' 
	ORDER BY uploaded_at ASC LIMIT 100;`
	rows, err := db.pool.Query(ctx, select100NewOrders)
	if err != nil {
		return nil, fmt.Errorf("failed to select new orders: %w", err)
	}
	defer rows.Close()

	return db.ScanOrders(rows)
}

func (db *DB) GetOrdersForUser(ctx context.Context, userID string) ([]models.OrderWithTime, error) {
	const selectOrdersForUser = `
	SELECT number, status, accrual, uploaded_at FROM users_orders WHERE user_id = $1 ORDER BY uploaded_at ASC;`
	rows, err := db.pool.Query(ctx, selectOrdersForUser, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to select by user_id: %w", err)
	}
	defer rows.Close()

	return db.ScanOrders(rows)
}

func (db *DB) ScanOrders(rows pgx.Rows) ([]models.OrderWithTime, error) {
	var orders []models.OrderWithTime
	var order models.OrderWithTime
	var timeDB string
	var err error
	for rows.Next() {
		if err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &timeDB); err != nil {
			db.log.Errorf("failed to get rows from select by user_id: %w", err)
		}
		order.UploadedAt, err = time.Parse(time.RFC3339, timeDB)
		if err != nil {
			db.log.Errorf("failed to parse date: %w", err)
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (db *DB) Getbalance(ctx context.Context, userID string) (models.Balance, error) {
	var balance models.Balance

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return balance, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			db.log.Infof("failed to rollback: %w", err)
		}
	}()

	balance, err = db.getBalanceDB(ctx, tx, userID)
	if err != nil {
		return balance, fmt.Errorf("failed to get balance from db: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return balance, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return balance, nil
}

func (db *DB) getBalanceDB(ctx context.Context, tx pgx.Tx, userID string) (models.Balance, error) {
	var balance models.Balance
	const selectSumAccrual = `SELECT SUM(accrual) FROM users_orders WHERE user_id = $1 GROUP BY user_id;`
	if err := tx.QueryRow(ctx, selectSumAccrual, userID).Scan(&balance.Current); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return balance, fmt.Errorf("failed to get withdraws sum: %w", err)
		}
	}

	const selectSumWithdrawn = `SELECT SUM(sum) FROM users_withdraw WHERE user_id = $1 GROUP BY user_id;`
	if err := tx.QueryRow(ctx, selectSumWithdrawn, userID).Scan(&balance.Withdrawn); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return balance, fmt.Errorf("failed to get withdraws sum: %w", err)
		}
	}

	balance.Current -= balance.Withdrawn

	return balance, nil
}

func (db *DB) SaveWithdraw(ctx context.Context, userID string, withdraw models.Withdraw) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			db.log.Infof("failed to rollback: %w", err)
		}
	}()

	balance, err := db.getBalanceDB(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("failed to get balance from db: %w", err)
	}

	if balance.Current < withdraw.Sum {
		return myErrors.ErrNoMoney
	}

	var orderDB string
	const selectOrder = `SELECT number FROM users_withdraw WHERE number = $1;`
	if err := tx.QueryRow(ctx, selectOrder, withdraw.Order).Scan(&orderDB); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to check withdraw in db: %w", err)
		}
	}

	if orderDB == withdraw.Order {
		return myErrors.ErrWithdrawAlreadySaved
	}

	currentTime := time.Now()
	rfc3339String := currentTime.Format(time.RFC3339)
	const insertWithdraw = `
	INSERT INTO users_withdraw (number, user_id, sum, processed_at) VALUES ($1, $2, $3, $4) RETURNING number;`
	var numberDB string
	if err := tx.QueryRow(
		ctx, insertWithdraw, withdraw.Order, userID, withdraw.Sum, rfc3339String).Scan(&numberDB); err != nil {
		return fmt.Errorf("failed to insert new user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (db *DB) GetWindrawalsForUser(ctx context.Context, userID string) ([]models.Withdrawals, error) {
	const selectWindrawalsForUser = `
	SELECT number, sum, processed_at FROM users_withdraw WHERE user_id = $1 ORDER BY processed_at ASC;`

	rows, err := db.pool.Query(ctx, selectWindrawalsForUser, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to select windrawals from db: %w", err)
	}
	defer rows.Close()

	var windrawals []models.Withdrawals
	for rows.Next() {
		var order, timeDB string
		var sum float64

		err := rows.Scan(&order, &sum, &timeDB)
		if err != nil {
			db.log.Errorf("failed to get rows from users_withdraw by user_id: %w", err)
		}
		var withdraw models.Withdrawals
		withdraw.Order = order
		withdraw.Sum = sum
		withdraw.ProcessedAt, err = time.Parse(time.RFC3339, timeDB)
		if err != nil {
			db.log.Errorf("failed to parse time: %w", err)
		}
		windrawals = append(windrawals, withdraw)
	}

	return windrawals, nil
}

func (db *DB) UpdateOrderAccrual(ctx context.Context, order models.Order) error {
	const updateSchemaDeletedFlag = `UPDATE users_orders SET accrual = $1, status = $2 WHERE number = $3;`

	_, err := db.pool.Exec(ctx, updateSchemaDeletedFlag, order.Accrual, order.Status, order.Order)
	if err != nil {
		return fmt.Errorf("failed to update order=%s: %w", order.Order, err)
	}
	return nil
}
