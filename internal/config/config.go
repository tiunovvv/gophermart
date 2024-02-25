package config

import (
	"flag"
	"os"
)

type Config struct {
	RunAddress           string
	DatabaseDSN          string
	AccrualSystemAddress string
}

func GetConfig() *Config {
	runAddress := flag.String("a", "localhost:8080", "runAddress")
	databaseDSN := flag.String("d", "", "databaseDSN")
	accrualSystemAddress := flag.String("r", "http://localhost:8000", "accrualSystemAddress")
	flag.Parse()

	config := Config{
		RunAddress:           getRunAddress(runAddress),
		DatabaseDSN:          getDatabaseURI(databaseDSN),
		AccrualSystemAddress: getAccrualSystemAddress(accrualSystemAddress),
	}

	return &config
}

func getRunAddress(runAddress *string) string {
	if envRunAddress, ok := os.LookupEnv("RUN_ADDRESS"); ok {
		return envRunAddress
	}

	return *runAddress
}

func getDatabaseURI(flagBaseURL *string) string {
	if envDatabaseURI, ok := os.LookupEnv("DATABASE_URI"); ok {
		return envDatabaseURI
	}

	return *flagBaseURL
}

func getAccrualSystemAddress(filePath *string) string {
	if envAccrualSystemAddress, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok {
		return envAccrualSystemAddress
	}

	return *filePath
}
