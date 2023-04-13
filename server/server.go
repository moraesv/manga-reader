package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	handler "manga-reader/server/handlers"
	"manga-reader/utils"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
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

	Url := vars.URL
	MangaUrl := vars.MANGA_URL

	if Url == "" {
		Url = "http://localhost"
	}

	Port, _ := strconv.Atoi(vars.PORT)
	if Port == 0 {
		Port = 5000
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", Port),
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
		Handler:      handlers.RecoveryHandler()(c.Handler(router)),
	}

	if vars.ENVIRONMENT == "DEV" {
		log.Printf("Iniciando servidor: %s:%d", Url, Port)

	} else {
		log.Printf("Iniciando servidor: %s", Url)
	}

	log.Printf("Utilizando: %s", MangaUrl)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}
}
