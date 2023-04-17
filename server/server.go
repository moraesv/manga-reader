package server

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	handler "manga-reader/server/handlers"
	"manga-reader/utils"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

// Server é o struct que define o server
type Server struct{}

// IniciaServidor
func (s *Server) IniciaServidor() {
	vars := utils.InitVars()
	uLogger := utils.NewGenericLogger()

	r := mux.NewRouter()

	c := cors.New(cors.Options{
		AllowedHeaders: []string{"X-Requested-With", "Content-Type", "Authorization"},
		AllowedMethods: []string{"GET", "POST", "DELETE", "PUT", "OPTIONS"},
		AllowedOrigins: []string{"*"},
	})

	handler.NewPage(r, vars)

	routes := r.PathPrefix("/api").Subrouter()
	handler.NewHealth(routes)
	handler.NewManga(routes, vars, uLogger)

	router := mux.NewRouter()
	router.Handle("/{_:.*}", r)

	Port, _ := strconv.Atoi(vars.PORT)
	if Port == 0 {
		Port = 9003
	}

	ctx := context.Background()
	tun, err := ngrok.Listen(ctx,
		config.HTTPEndpoint(),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	vars.URL = tun.URL()
	vars.URL_DOWNLOAD_MANGA = vars.URL + "/api/manga"

	SalvaNovaURL(vars.URL_REDIRECT_FULL, vars.URL)

	uLogger.LogIt("INFO", fmt.Sprintf("Utilizando: %s", vars.MANGA_URL), nil)
	uLogger.LogIt("INFO", fmt.Sprintf("Iniciando servidor: %s", vars.URL), nil)

	http.Serve(tun, handlers.RecoveryHandler()(c.Handler(router)))
}

func SalvaNovaURL(urlRedirect, urlNgrok string) {
	uLogger := utils.NewGenericLogger()
	url := urlRedirect + "/url"
	body := []byte(urlNgrok)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao criar a requisição: %s", err.Error()), nil)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao enviar a requisição: %s", err.Error()), nil)
		return
	}
	defer resp.Body.Close()

	uLogger.LogIt("INFO", fmt.Sprintf("Resposta do servidor: %s", resp.Status), nil)
}
