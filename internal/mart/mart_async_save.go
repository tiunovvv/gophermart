package mart

import (
	"context"

	"github.com/tiunovvv/gophermart/internal/database"
	"go.uber.org/zap"
)

type Job struct {
	userID string
	number string
}

type Worker struct {
	jobQueue chan Job
	logger   *zap.Logger
	database database.DB
	id       int
}

type Dispatcher struct {
	jobQueue   chan Job
	workerPool []*Worker
}

func NewWorker(id int, jobQueue chan Job, database database.DB, logger *zap.Logger) *Worker {
	return &Worker{
		id:       id,
		jobQueue: jobQueue,
		database: database,
		logger:   logger,
	}
}

func NewDispatcher(
	ctx context.Context,
	workerCount int,
	jobQueue chan Job,
	database database.DB,
	logger *zap.Logger) *Dispatcher {
	workerPool := make([]*Worker, workerCount)
	for i := 0; i < workerCount; i++ {
		worker := NewWorker(i, jobQueue, database, logger)
		workerPool[i] = worker
		go worker.Start(ctx)
	}

	return &Dispatcher{
		workerPool: workerPool,
		jobQueue:   jobQueue,
	}
}

func (w *Worker) Start(ctx context.Context) {
	for job := range w.jobQueue {
		if err := w.database.SaveOrder(ctx, job.userID, job.number); err != nil {
			w.logger.Sugar().Errorf("failed to save order: %w", err)
		}
	}
}
