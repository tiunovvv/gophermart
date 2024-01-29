package mart

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/tiunovvv/gophermart/internal/database"
	"github.com/tiunovvv/gophermart/internal/models"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Mart struct {
	database *database.DB
	logger   *zap.Logger
}

func NewMart(database *database.DB, logger *zap.Logger) *Mart {
	return &Mart{
		database: database,
		logger:   logger,
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

	err = m.database.NewUser(ctx, userID, user.Login, string(hash))
	if err != nil {
		return "", fmt.Errorf("failed to save new user: %w", err)
	}
	return userID, nil
}

func (m *Mart) GetUserID(ctx context.Context, user models.User) (string, error) {
	userID, hash, err := m.database.GetUserID(ctx, user.Login)
	if err != nil {
		return "", fmt.Errorf("failed to get data for user from db: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(user.Password)); err != nil {
		return "", fmt.Errorf("failed to check password: %w", err)
	}

	return userID, nil
}

func (m *Mart) GetUserIDForOrder(ctx context.Context, number string) (string, error) {
	userID, err := m.database.GetUserIDForOrder(ctx, number)
	if err != nil {
		return "", fmt.Errorf("failed to get user ID: %w", err)
	}
	return userID, nil
}

func (m *Mart) SaveOrder(ctx context.Context, userID string, number string) {
	const countOfWorkers = 1
	jobQueue := make(chan Job, countOfWorkers)
	dispatcher := NewDispatcher(ctx, countOfWorkers, jobQueue, *m.database, m.logger)

	go func() {
		var wg sync.WaitGroup
		dispatcher.jobQueue <- Job{userID: userID, number: number}
		close(dispatcher.jobQueue)
		for _, worker := range dispatcher.workerPool {
			wg.Add(1)
			go func(w *Worker) {
				defer wg.Done()
				w.Start(ctx)
			}(worker)
		}
		wg.Wait()
	}()
}

func (m *Mart) GetNumbersForUser(ctx context.Context, userID string) (map[string]time.Time, error) {
	orders, err := m.database.GetNumbersForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to orders for user: %w", err)
	}
	return orders, nil
}

func (m *Mart) GetOrderInfo(address string, number string) (models.OrderWithTime, error) {
	return m.GetOrder(address, number)
}

func (m *Mart) GetOrder(address string, number string) (models.OrderWithTime, error) {
	var order models.OrderWithTime
	url, err := url.JoinPath(address, "/api/orders/", number)
	if err != nil {
		return order, fmt.Errorf("failed to join path: %w", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return order, fmt.Errorf("failed to get data from accrual: %w", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			m.logger.Sugar().Errorf("failed to close body: %w", err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return order, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) == 0 {
		order.Number = number
		order.Status = `NEW`
		return order, nil
	}

	if err := json.Unmarshal(body, &order); err != nil {
		return order, fmt.Errorf("failed to decode JSON: %w", err)
	}
	return order, nil
}

func (m *Mart) GetSumOrders(ctx context.Context, address string, userID string) (float64, error) {
	orders, err := m.database.GetNumbersForUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get orders for user: %w", err)
	}

	var sum float64
	for order := range orders {
		orderDetails, err := m.GetOrder(address, order)
		if err != nil {
			m.logger.Sugar().Infof("failed to det data from accrual for: %s", order)
			continue
		}
		sum += orderDetails.Accrual
	}
	return sum, nil
}

func (m *Mart) GetSumWithdrawn(ctx context.Context, userID string) (float64, error) {
	withdraw, err := m.database.GetSumWithdrawn(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to det withdraw sum: %w", err)
	}
	return withdraw, nil
}

func (m *Mart) SaveWithdraw(ctx context.Context, userID string, order string, sum float64) error {
	if err := m.database.SaveWithdraw(ctx, userID, order, sum); err != nil {
		return fmt.Errorf("failed to save withdraw: %w", err)
	}
	return nil
}

func (m *Mart) GetWindrawalsForUser(ctx context.Context, userID string) ([]models.Withdrawals, error) {
	withdrawals, err := m.database.GetWindrawalsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawals: %w", err)
	}
	return withdrawals, nil
}
