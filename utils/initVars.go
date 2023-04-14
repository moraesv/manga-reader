package utils

import (
	"manga-reader/models"
	"os"
)

// Seta os valores das envs
func InitVars() *models.Vars {
	v := models.Vars{
		URL:                os.Getenv("URL"),
		PORT:               os.Getenv("PORT"),
		ENVIRONMENT:        os.Getenv("ENVIRONMENT"),
		LOGRUS_LOG_LEVEL:   os.Getenv("LOGRUS_LOG_LEVEL"),
		MANGA_URL:          os.Getenv("MANGA_URL"),
		URL_DOWNLOAD_MANGA: os.Getenv("URL_DOWNLOAD_MANGA"),
		URL_REDIRECT:       os.Getenv("URL_REDIRECT"),
		PORT_REDIRECT:      os.Getenv("PORT_REDIRECT"),
		URL_REDIRECT_FULL:  os.Getenv("URL_REDIRECT_FULL"),
	}

	return &v
}
