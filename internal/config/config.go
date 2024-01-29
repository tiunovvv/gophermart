package config

import (
	"flag"
	"os"

	"go.uber.org/zap"
)

type Config struct {
	logger               *zap.Logger
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
}

func NewConfig(logger *zap.Logger) *Config {
	runAddress := flag.String("a", "localhost:8080", "runAddress")
	databaseURI := flag.String("d", "", "databaseURI")
	accrualSystemAddress := flag.String("r", "http://localhost:8000", "accrualSystemAddress")
	flag.Parse()

	config := Config{
		logger:               logger,
		RunAddress:           getRunAddress(runAddress),
		DatabaseURI:          getDatabaseURI(databaseURI),
		AccrualSystemAddress: getAccrualSystemAddress(accrualSystemAddress),
	}

	logger.Sugar().Infof("runAddress: %s", config.RunAddress)
	logger.Sugar().Infof("databaseURI: %s", config.DatabaseURI)
	logger.Sugar().Infof("accrualSystemAddress: %s", config.AccrualSystemAddress)

	return &config
}

func getRunAddress(runAddress *string) string {
	if envRunAddress := os.Getenv("RUN_ADDRESS"); len(envRunAddress) != 0 {
		return envRunAddress
	}

	return *runAddress
}

func getDatabaseURI(flagBaseURL *string) string {
	if envDatabaseURI := os.Getenv("DATABASE_URI"); len(envDatabaseURI) != 0 {
		return envDatabaseURI
	}

	return *flagBaseURL
}

func getAccrualSystemAddress(filePath *string) string {
	if envAccrualSystemAddress := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrualSystemAddress != "" {
		return envAccrualSystemAddress
	}

	return *filePath
}
