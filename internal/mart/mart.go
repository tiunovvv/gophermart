package mart

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/tiunovvv/gophermart/internal/database"
	"github.com/tiunovvv/gophermart/internal/models"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Mart struct {
	db  *database.DB
	log *zap.SugaredLogger
}

func NewMart(db *database.DB, log *zap.SugaredLogger) *Mart {
	return &Mart{
		db:  db,
		log: log,
	}
}

func (m *Mart) NewUser(ctx context.Context, user models.User) (string, error) {
	uuid, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("failed to create uuid: %w", err)
	}

	userID := uuid.String()

	const length = 10
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), length)
	if err != nil {
		return "", fmt.Errorf("failed to create password hash: %w", err)
	}

	err = m.db.NewUser(ctx, userID, user.Login, string(hash))
	if err != nil {
		return "", fmt.Errorf("failed to save new user: %w", err)
	}
	return userID, nil
}

func (m *Mart) GetUserID(ctx context.Context, user models.User) (string, error) {
	userID, hash, err := m.db.GetUserID(ctx, user.Login)
	if err != nil {
		return "", fmt.Errorf("failed to get data for user from db: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(user.Password)); err != nil {
		return "", fmt.Errorf("failed to check password: %w", err)
	}

	return userID, nil
}

func (m *Mart) SaveOrder(ctx context.Context, userID string, number string) error {
	err := m.db.SaveOrder(ctx, userID, number)
	if err != nil {
		m.log.Errorf("failed to save order: %v", err)
		return fmt.Errorf("failed to save order: %w", err)
	}
	return nil
}

func (m *Mart) GetNewOrders(ctx context.Context) ([]models.OrderWithTime, error) {
	orders, err := m.db.GetNewOrders(ctx)
	if err != nil {
		m.log.Errorf("failed to get new orders: %v", err)
		return nil, fmt.Errorf("failed to get new orders: %w", err)
	}
	return orders, nil
}

func (m *Mart) GetOrdersForUser(ctx context.Context, userID string) ([]models.OrderWithTime, error) {
	orders, err := m.db.GetOrdersForUser(ctx, userID)
	if err != nil {
		m.log.Errorf("failed to get orders for user: %v", err)
		return nil, fmt.Errorf("failed to get orders for user: %w", err)
	}
	return orders, nil
}

func (m *Mart) GetBalance(ctx context.Context, userID string) (models.Balance, error) {
	balance, err := m.db.Getbalance(ctx, userID)
	if err != nil {
		m.log.Errorf("failed to get balance for user: %v", err)
		return balance, fmt.Errorf("failed to get balance for user: %w", err)
	}
	return balance, nil
}

func (m *Mart) SaveWithdraw(ctx context.Context, userID string, withdraw models.Withdraw) error {
	err := m.db.SaveWithdraw(ctx, userID, withdraw)
	if err != nil {
		m.log.Errorf("failed to save withdraw: %v", err)
		return fmt.Errorf("failed to save withdraw: %w", err)
	}
	return nil
}

func (m *Mart) GetWindrawalsForUser(ctx context.Context, userID string) ([]models.Withdrawals, error) {
	withdrawals, err := m.db.GetWindrawalsForUser(ctx, userID)
	if err != nil {
		m.log.Errorf("failed to get withdrawals: %v", err)
		return nil, fmt.Errorf("failed to get withdrawals: %w", err)
	}
	return withdrawals, nil
}

func (m *Mart) UpdateOrderAccrual(ctx context.Context, order models.Order) error {
	err := m.db.UpdateOrderAccrual(ctx, order)
	if err != nil {
		m.log.Errorf("failed to update order: %v", err)
		return fmt.Errorf("failed to update order: %w", err)
	}
	return nil
}
