package server

import (
	"manga-reader/models"
	"net/http"
	"text/template"

	"github.com/gorilla/mux"
)

type PageHandler struct {
	vars *models.Vars
}

type Modelo struct {
	UrlDownloadManga string
	UrlSiteManga     string
}

func NewPage(e *mux.Router, vars *models.Vars) {
	handler := PageHandler{vars}

	s := e.PathPrefix("/").Subrouter()
	s.HandleFunc("/", handler.Index).Methods("GET")
}

func (p *PageHandler) Index(writer http.ResponseWriter, request *http.Request) {
	UrlDownloadManga := p.vars.URL_DOWNLOAD_MANGA
	UrlSiteManga := p.vars.MANGA_URL

	modelo := Modelo{UrlDownloadManga, UrlSiteManga}

	templateHTML, _ := template.ParseFiles("./index.html")

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := templateHTML.Execute(writer, modelo)
	if err != nil {
		http.Error(writer, "Erro ao renderizar arquivo HTML", http.StatusInternalServerError)
		return
	}
}
