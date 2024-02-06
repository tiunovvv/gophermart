package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/tiunovvv/gophermart/internal/config"
	"github.com/tiunovvv/gophermart/internal/mart"
	"github.com/tiunovvv/gophermart/internal/models"
	"go.uber.org/zap"
)

type Worker struct {
	OrdersChan chan models.OrderWithTime
	ErrorChan  chan accrualError
	ID         int
}

func (w *Worker) Start(
	ctx context.Context,
	wg *sync.WaitGroup,
	cfg *config.Config,
	log *zap.SugaredLogger,
	mart *mart.Mart,
) {
	defer wg.Done()
	for order := range w.OrdersChan {
		order, errAcc := w.getOrder(log, cfg.AccrualSystemAddress, order.Number)

		if errors.Is(errAcc.error, errTooManyRequests) {
			log.Errorf("failed get info about order %s from accrual: %w", order.Order, errAcc.error)
			w.ErrorChan <- errAcc
			return
		}

		if errAcc.error != nil {
			log.Errorf("failed get info about order %s from accrual: %w", order.Order, errAcc.error)
			return
		}

		err := mart.UpdateOrderAccrual(ctx, order)
		if err != nil {
			log.Errorf("failed to update order %s", order.Order)
			return
		}
	}
}

func (w *Worker) getOrder(log *zap.SugaredLogger, address string, number string) (models.Order, accrualError) {
	var order models.Order
	url, err := url.JoinPath(address, "/api/orders/", number)
	if err != nil {
		return order, accrualError{error: fmt.Errorf("failed to join path: %w", err), timeout: 0}
	}

	resp, err := http.Get(url)
	if err != nil {
		return order, accrualError{error: fmt.Errorf("failed to get request from accural: %w", err), timeout: 0}
	}

	if resp.StatusCode == http.StatusNoContent {
		return order, accrualError{error: errOrderNotRegistered, timeout: 0}
	}

	if resp.StatusCode == http.StatusInternalServerError {
		return order, accrualError{error: errAccrualServerError, timeout: 0}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		timeout, err := strconv.Atoi(retryAfter)
		if err != nil {
			return order, accrualError{error: fmt.Errorf("failed to convert timeout: %w", err), timeout: 0}
		}

		return order, accrualError{error: errTooManyRequests, timeout: timeout}
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Errorf("failed to close body: %w", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return order, accrualError{error: fmt.Errorf("failed to read body: %w", err), timeout: 0}
	}

	if err := json.Unmarshal(body, &order); err != nil {
		return order, accrualError{error: fmt.Errorf("failed to unmarshal body: %w", err), timeout: 0}
	}
	return order, accrualError{error: nil, timeout: 0}
}
