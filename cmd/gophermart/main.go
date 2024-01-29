package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tiunovvv/gophermart/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	server, err := server.NewServer(ctx)
	if err != nil {
		return fmt.Errorf("error building server: %w", err)
	}

	if err := server.Start(); err != nil {
		return fmt.Errorf("error starting server: %w", err)
	}

	return nil
}
