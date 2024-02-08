package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/tiunovvv/gophermart/internal/accrual"
	"github.com/tiunovvv/gophermart/internal/config"
	"github.com/tiunovvv/gophermart/internal/database"
	"github.com/tiunovvv/gophermart/internal/handler"
	"github.com/tiunovvv/gophermart/internal/mart"
	"github.com/tiunovvv/gophermart/internal/server"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

const (
	timeoutServerShutdown = time.Second * 5
	timeoutShutdown       = time.Second * 10
)

func run() error {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelCtx()

	context.AfterFunc(ctx, func() {
		ctx, cancelCtx := context.WithTimeout(context.Background(), timeoutShutdown)
		defer cancelCtx()

		<-ctx.Done()
		log.Fatal("failed to gracefully shutdown the service")
	})

	wg := &sync.WaitGroup{}
	defer func() {
		wg.Wait()
	}()

	cfg := config.GetConfig()

	logger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to initialize logger %w", err)
	}

	log := logger.Sugar()

	db, err := database.NewDB(ctx, cfg.DatabaseDSN, log)
	if err != nil {
		return fmt.Errorf("failed to initialize a new DB %w", err)
	}

	mart := mart.NewMart(db, log)

	const workerCount = 3
	disp := accrual.NewDispatcher(cfg, mart, log, workerCount)
	go disp.Start(ctx)

	watch(ctx, wg, db)

	h := handler.NewHandler(cfg, mart, log)
	srv := server.InitServer(h, cfg, logger)

	componentsErrs := make(chan error, 1)

	manageServer(ctx, wg, srv, componentsErrs)

	select {
	case <-ctx.Done():
	case err := <-componentsErrs:
		log.Error(err)
		cancelCtx()
	}

	return nil
}

func watch(ctx context.Context, wg *sync.WaitGroup, db *database.DB) {
	wg.Add(1)
	go func() {
		defer log.Print("closed DB and stoped Dispatcher")
		defer wg.Done()

		<-ctx.Done()

		db.Close()
	}()
}

func manageServer(ctx context.Context, wg *sync.WaitGroup, srv *http.Server, errs chan<- error) {
	go func(errs chan<- error) {
		if err := srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			errs <- fmt.Errorf("listen and server has failed: %w", err)
		}
	}(errs)

	wg.Add(1)
	go func() {
		defer log.Print("server has been shutdown")
		defer wg.Done()
		<-ctx.Done()

		shutdownTimeoutCtx, cancelShutdownTimeoutCtx := context.WithTimeout(context.Background(), timeoutServerShutdown)
		defer cancelShutdownTimeoutCtx()
		if err := srv.Shutdown(shutdownTimeoutCtx); err != nil {
			log.Printf("an error occurred during server shutdown: %v", err)
		}
	}()
}
