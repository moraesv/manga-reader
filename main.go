package main

import (
	"time"

	"manga-reader/server"
	"manga-reader/utils"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	utilsLogger := utils.NewGenericLogger()

	utilsLogger.LogIt("INFO", "Iniciando API Manga Reader...", nil)
	utilsLogger.LogIt("INFO", "Em: "+time.Now().Format("02/01/2006 15:04:05"), nil)

	api := server.Server{}
	api.IniciaServidor()
}
