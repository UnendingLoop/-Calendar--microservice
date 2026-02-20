// Package config reads port from env-file and uses default value if port not specified
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func GetEnvs() (port, filename string) {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	port = os.Getenv("APP_PORT")
	if port == "" {
		log.Println("APP_PORT is not set in env. Using default value 8080")
		port = "8080"
	}

	filename = os.Getenv("FILENAME")
	if filename == "" {
		log.Println("FILENAME is not set in env. Using default value storage.txt")
		filename = "storage.txt"
	}
	log.Printf("Fetched config from env:\nSourcefile for eventsmap: %q\nApp port: %v\n", filename, port)
	return
}
