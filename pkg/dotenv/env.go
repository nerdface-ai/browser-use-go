package dotenv

import (
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

func LoadEnv(envPath string) {
	err := godotenv.Load(envPath)
	if err != nil {
		log.Debug(err.Error())
		log.Debug("Error loading .env file")
	}
}
