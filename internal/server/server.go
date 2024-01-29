package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/tiunovvv/gophermart/internal/config"
	"github.com/tiunovvv/gophermart/internal/database"
	"github.com/tiunovvv/gophermart/internal/handler"
	"github.com/tiunovvv/gophermart/internal/mart"

	"go.uber.org/zap"
)

type Server struct {
	logger   *zap.Logger
	database *database.DB
	*http.Server
}

func NewServer(ctx context.Context) (*Server, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	config := config.NewConfig(logger)
	database, err := database.NewDB(ctx, config.DatabaseURI, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	mart := mart.NewMart(database, logger)
	handler := handler.NewHandler(config, mart, logger)

	errorLog := zap.NewStdLog(logger)

	const (
		bytes   = 20
		seconds = 5 * time.Second
	)

	s := http.Server{
		Addr:           config.RunAddress,
		Handler:        handler.InitRoutes(),
		ErrorLog:       errorLog,
		MaxHeaderBytes: 1 << bytes,
		ReadTimeout:    seconds,
		WriteTimeout:   seconds,
	}

	return &Server{logger, database, &s}, nil
}

func (s *Server) Start() error {
	var err error
	defer func() {
		if er := s.logger.Sync(); er != nil {
			err = fmt.Errorf("failed to sync logger: %w", er)
		}
	}()

	defer func() {
		if er := s.database.Close(); er != nil {
			err = fmt.Errorf("failed to close store: %w", er)
		}
	}()

	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("could not listen on", zap.String("addr", s.Addr), zap.Error(err))
		}
	}()

	s.logger.Info("server is ready to handle requests", zap.String("addr", s.Addr))
	s.gracefulShutdown()
	return err
}

func (s *Server) gracefulShutdown() {
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, os.Interrupt)
	sig := <-quit
	s.logger.Info("server is shutting down", zap.String("reason", sig.String()))
	const seconds = 10 * time.Second
	ctx, cancelCtx := context.WithTimeout(context.Background(), seconds)
	defer cancelCtx()

	s.SetKeepAlivesEnabled(false)
	if err := s.Shutdown(ctx); err != nil {
		s.logger.Error("failed to gracefully shutdown the server", zap.Error(err))
	}
	s.logger.Info("server is stopped")
}
