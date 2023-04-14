package server

import (
	"context"
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

	log.Printf("Utilizando: %s", vars.MANGA_URL)

	log.Printf("Iniciando servidor: %s", vars.URL)

	http.Serve(tun, handlers.RecoveryHandler()(c.Handler(router)))
}
