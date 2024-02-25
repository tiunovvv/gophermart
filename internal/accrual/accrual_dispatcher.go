package accrual

import (
	"context"
	"sync"
	"time"

	"github.com/tiunovvv/gophermart/internal/config"
	"github.com/tiunovvv/gophermart/internal/mart"
	"github.com/tiunovvv/gophermart/internal/models"
	"go.uber.org/zap"
)

type accrualError struct {
	error
	timeout int
}

type Dispatcher struct {
	cfg         *config.Config
	mart        *mart.Mart
	log         *zap.SugaredLogger
	ordersChan  chan models.OrderWithTime
	errorChan   chan accrualError
	workerCount int
}

func NewDispatcher(cfg *config.Config, mart *mart.Mart, log *zap.SugaredLogger, workerCount int) *Dispatcher {
	dispatcher := &Dispatcher{
		cfg:         cfg,
		mart:        mart,
		log:         log,
		workerCount: workerCount,
	}

	return dispatcher
}

func (d *Dispatcher) Start(ctx context.Context) {
	d.ordersChan = make(chan models.OrderWithTime, d.workerCount)
	d.errorChan = make(chan accrualError)
	defer close(d.ordersChan)
	defer close(d.errorChan)

	var wg sync.WaitGroup
	for i := 1; i <= d.workerCount; i++ {
		worker := &Worker{
			ID:         i,
			OrdersChan: d.ordersChan,
			ErrorChan:  d.errorChan,
		}
		wg.Add(1)
		go worker.Start(ctx, &wg, d.cfg, d.log, d.mart)
	}

	go func() {
		for {
			orders, err := d.mart.GetNewOrders(ctx)
			if err != nil {
				d.log.Errorf("failed to get new orders: %v", err)
				continue
			}

			for _, order := range orders {
				d.ordersChan <- order
			}

			select {
			case err := <-d.errorChan:
				d.log.Errorf("pausing workers for %s seconds\n", err.timeout)
				pauseCtx, cancel := context.WithTimeout(ctx, time.Duration(err.timeout)*time.Second)
				pauseCtx.Done()
				cancel()
			default:
			}
			time.Sleep(time.Second)
		}
	}()
	wg.Wait()
}
