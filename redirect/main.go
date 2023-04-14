package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"manga-reader/utils"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	vars := utils.InitVars()

	http.HandleFunc("/url", handleURL)
	http.HandleFunc("/", handleRedirect)
	log.Printf("Iniciando servidor de redirecionamento: %s", vars.URL_REDIRECT_FULL)

	http.ListenAndServe(fmt.Sprintf(":%s", vars.PORT_REDIRECT), nil)
}

func handleURL(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Erro ao ler o corpo da requisição", http.StatusBadRequest)
			return
		}
		err = ioutil.WriteFile("ngrok-url.txt", body, 0644)
		if err != nil {
			http.Error(w, "Erro ao salvar a URL do ngrok em arquivo", http.StatusInternalServerError)
			return
		}
		log.Println("URL do ngrok salva com sucesso!")
		fmt.Fprintf(w, "URL do ngrok salva com sucesso!")
	} else {
		log.Println("Erro ao salvar URL do ngrok!")
		http.Error(w, "Método HTTP inválido. Use PUT para salvar a URL do ngrok.", http.StatusMethodNotAllowed)
	}
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	ngrokURLBytes, err := ioutil.ReadFile("ngrok-url.txt")
	if err != nil {
		fmt.Fprintf(w, "Nenhuma URL do servidor salva ainda.")
		return
	}
	ngrokURL := string(ngrokURLBytes)

	resp, err := http.Get(ngrokURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Fprintf(w, "Servidor desligado.")
		return
	}

	http.Redirect(w, r, ngrokURL, http.StatusTemporaryRedirect)
}
