package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI      string
	MongoDBName   string
	BotToken      string
	SleepTime     int
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("ℹ️ No .env file found, reading from environment variables")
	}

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		return nil, fmt.Errorf("missing BOT_TOKEN in environment variables")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return nil, fmt.Errorf("missing MONGO_URI in environment variables")
	}

	sleepTimeStr := getEnv("SLEEP_TIME", "300")
	sleepTime, err := strconv.Atoi(sleepTimeStr)
	if err != nil {
		log.Printf("⚠️ Invalid SLEEP_TIME '%s', defaulting to 300s\n", sleepTimeStr)
		sleepTime = 300
	}

	return &Config{
		MongoURI:    mongoURI,
		MongoDBName: getEnv("MONGO_DB_NAME", "testflight_checker"),
		BotToken:    botToken,
		SleepTime:   sleepTime,
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
