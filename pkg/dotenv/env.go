package dotenv

import (
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

func LoadEnv(envPath string) {
	err := godotenv.Load(envPath)
	if err != nil {
		log.Print(err)
		log.Debug("Error loading .env file")
	}
}
