package server

import (
	"encoding/json"
	"manga-reader/models"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func NewHealth(e *mux.Router) {
	s := e.PathPrefix("/").Subrouter()
	s.HandleFunc("/health", Health).Methods("GET")
}

func Health(writer http.ResponseWriter, request *http.Request) {
	health := models.Health{}

	health.Status = "up"
	health.Time = time.Now()

	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(health)
}
