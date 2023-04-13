package server

import (
	"net/http"
	"os"
	"text/template"

	"github.com/gorilla/mux"
)

type Modelo struct {
	UrlDownloadManga string
	UrlSiteManga     string
}

func NewPage(e *mux.Router) {
	s := e.PathPrefix("/").Subrouter()
	s.HandleFunc("/", Index).Methods("GET")
}

func Index(writer http.ResponseWriter, request *http.Request) {
	UrlDownloadManga := os.Getenv("URL_DOWNLOAD_MANGA")
	UrlSiteManga := os.Getenv("MANGA_URL")

	modelo := Modelo{UrlDownloadManga, UrlSiteManga}

	templateHTML, _ := template.ParseFiles("./index.html")

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := templateHTML.Execute(writer, modelo)
	if err != nil {
		http.Error(writer, "Erro ao renderizar arquivo HTML", http.StatusInternalServerError)
		return
	}
}
