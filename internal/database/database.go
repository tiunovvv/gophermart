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
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewDB(ctx context.Context, databaseURI string, logger *zap.Logger) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(databaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the DSN: %w", err)
	}

	queryTracer := NewQueryTracer(logger)
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

	database := &DB{pool: pool, logger: logger}
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

func (db *DB) Close() error {
	db.pool.Close()
	return nil
}

func (db *DB) NewUser(ctx context.Context, userID string, login string, hash string) error {
	const insertUser = `INSERT INTO users (user_id, login, pswd_hash) VALUES ($1, $2, $3)`
	var loginDB string
	err := db.pool.QueryRow(ctx, insertUser, userID, login, hash).Scan(&loginDB)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return myErrors.ErrLoginAlreadySaved
		}
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

func (db *DB) GetUserIDForOrder(ctx context.Context, number string) (string, error) {
	const selectOrder = `SELECT user_id FROM users_orders WHERE number = $1;`
	row := db.pool.QueryRow(ctx, selectOrder, number)
	var userID string
	if err := row.Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to select order from db: %w", err)
	}
	return userID, nil
}

func (db *DB) SaveOrder(ctx context.Context, userID string, number string) error {
	currentTime := time.Now()
	rfc3339String := currentTime.Format(time.RFC3339)
	const insertOrder = `INSERT INTO users_orders (number, user_id, uploaded_at) VALUES ($1, $2, $3)`
	var orderDB string
	err := db.pool.QueryRow(ctx, insertOrder, number, userID, rfc3339String).Scan(&orderDB)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return myErrors.ErrWithdrawAlreadySaved
		}
	}
	return nil
}

func (db *DB) GetNumbersForUser(ctx context.Context, userID string) (map[string]time.Time, error) {
	const selectOrdersForUser = `SELECT number, uploaded_at FROM users_orders WHERE user_id = $1 ORDER BY uploaded_at ASC;`
	rows, err := db.pool.Query(ctx, selectOrdersForUser, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to select by user_id: %w", err)
	}
	defer rows.Close()

	numbers := make(map[string]time.Time)
	for rows.Next() {
		var number, timeDB string
		if err := rows.Scan(&number, &timeDB); err != nil {
			db.logger.Sugar().Errorf("failed to get rows from select by user_id: %w", err)
		}
		numbers[number], err = time.Parse(time.RFC3339, timeDB)
		if err != nil {
			db.logger.Sugar().Errorf("failed to parse date: %w", err)
		}
	}
	return numbers, nil
}

func (db *DB) GetSumWithdrawn(ctx context.Context, userID string) (float64, error) {
	const selectSumWithdrawn = `SELECT SUM(sum) FROM users_withdraw WHERE user_id = $1 GROUP BY user_id;;`
	row := db.pool.QueryRow(ctx, selectSumWithdrawn, userID)
	var sum float64
	if err := row.Scan(&sum); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to select order from db: %w", err)
	}
	return sum, nil
}

func (db *DB) SaveWithdraw(ctx context.Context, userID string, order string, sum float64) error {
	currentTime := time.Now()
	rfc3339String := currentTime.Format(time.RFC3339)
	const insertWithdraw = `INSERT INTO users_withdraw (number, user_id, sum, processed_at) VALUES ($1, $2, $3, $4)`
	var orderDB string
	err := db.pool.QueryRow(ctx, insertWithdraw, order, userID, sum, rfc3339String).Scan(&orderDB)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return myErrors.ErrWithdrawAlreadySaved
		}
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
			db.logger.Sugar().Errorf("failed to get rows from users_withdraw by user_id: %w", err)
		}
		var withdraw models.Withdrawals
		withdraw.Order = order
		withdraw.Sum = sum
		withdraw.ProcessedAt, err = time.Parse(time.RFC3339, timeDB)
		if err != nil {
			db.logger.Sugar().Errorf("failed to parse time: %w", err)
		}
		windrawals = append(windrawals, withdraw)
	}

	return windrawals, nil
}
