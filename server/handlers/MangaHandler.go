package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"io/ioutil"
	"manga-reader/models"
	"manga-reader/utils"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jung-kurt/gofpdf"
	"github.com/pixiv/go-libjpeg/jpeg"
)

type MangaHandler struct {
	vars    *models.Vars
	uLogger utils.ILoggerUtils
}

func NewManga(e *mux.Router, vars *models.Vars, uLogger utils.ILoggerUtils) {

	handler := MangaHandler{vars, uLogger}

	s := e.PathPrefix("/").Subrouter()
	s.HandleFunc("/manga", handler.DownloadManga).Methods("POST")
}

func (m *MangaHandler) response(writer http.ResponseWriter, statusCode int, message string) {
	writer.WriteHeader(500)
	resp := make(map[string]string)
	resp["message"] = message
	jsonResp, _ := json.Marshal(resp)
	writer.Write(jsonResp)
}

func (m *MangaHandler) DownloadManga(writer http.ResponseWriter, request *http.Request) {
	uuidManga := uuid.New().String()

	m.uLogger.LogIt("DEBUG", "Iniciado DownloadManga ID: "+uuidManga, nil)

	// Decodifica o corpo da requisição
	var body models.BodyManga
	err := json.NewDecoder(request.Body).Decode(&body)
	if err != nil {
		m.response(writer, http.StatusBadRequest, fmt.Sprintf("Erro ao decodificar JSON: %s", err.Error()))
		return
	}

	err = m.TestaConexao()
	if err != nil {
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao TestaConexao ID: %s Error: %s", uuidManga, err.Error()), nil)
		m.response(writer, http.StatusInternalServerError, err.Error())
		return
	}

	uris, err := m.BuscaListaCapitulos(body.ID, body.CapInicio, body.CapFim)
	if err != nil {
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao BuscaListaCapitulos ID: %s Error: %s", uuidManga, err.Error()), nil)
		m.response(writer, http.StatusInternalServerError, err.Error())
		return
	}

	err = os.MkdirAll("./mangas/"+uuidManga, os.ModePerm)
	if err != nil {
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao criar diretório ID: %s Error: %s", uuidManga, err.Error()), nil)
		m.response(writer, http.StatusInternalServerError, err.Error())
		return
	}

	pdf, err := m.GerarManga(uuidManga, uris)
	if err != nil {
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao GerarManga ID: %s Error: %s", uuidManga, err.Error()), nil)
		m.response(writer, http.StatusInternalServerError, err.Error())
		return
	}

	writer.Header().Set("Content-Disposition", "attachment")
	writer.Header().Set("Content-Filename", "documento.pdf")
	writer.Header().Set("Content-Type", "application/pdf")
	writer.Header().Set("Content-Disposition", "attachment; filename=manga.pdf")

	pdf.Output(writer)

	m.uLogger.LogIt("DEBUG", "PDF enviado", nil)

	err = os.RemoveAll("./mangas/" + uuidManga)
	if err != nil {
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao excluir diretório ID: %s Error: %s", uuidManga, err.Error()), nil)
		return
	}
}

func (m *MangaHandler) GerarManga(uuidManga string, uris []string) (*gofpdf.Fpdf, error) {

	m.uLogger.LogIt("DEBUG", "Iniciado Gerar Manga ID: "+uuidManga, nil)

	c := make(chan bool, len(uris))

	for index, uri := range uris {
		go m.BuscarImagensCap(uuidManga, uri, index+1, c)
	}

	for range uris {
		<-c
	}

	pdf, err := m.GerarPDF(uuidManga)

	return pdf, err
}

func (m *MangaHandler) BuscarImagensCap(uuidManga string, uri string, index int, c chan (bool)) (*gofpdf.Fpdf, error) {
	count := 0
	m.uLogger.LogIt("DEBUG", fmt.Sprintf("Iniciado Gerar Manga ID: %s Capítulo %s", uuidManga, uri), nil)

	// URL da página web que queremos acessar
	url := m.vars.MANGA_URL + uri

	// Cria um contexto de execução para o navegador Chrome headless
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Navega até a página web e espera carregar totalmente
	var html string
	err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.Location(&url),
		chromedp.WaitVisible("div.reader-content div.manga-image img", chromedp.ByQuery),
		chromedp.InnerHTML("html", &html),
	})
	if err != nil {
		message := fmt.Sprintf("Erro ao navegar até a página web: %s", err.Error())
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
		return nil, err
	}

	// Clica na div "page-next" e baixa a próxima imagem até que não exista mais uma imagem na tela
	for {
		// Faz o download da imagem
		document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			message := fmt.Sprintf("Erro ao parsear o HTML da página: %s", err.Error())
			m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
			return nil, err
		}
		imageURL, _ := document.Find("div.reader-content div.manga-image img").First().Attr("src")

		if imageURL != "" {
			count += 1
			imageName := strings.Split(imageURL, "/")
			filename := fmt.Sprintf("%d.%d%s", index, count, filepath.Ext(imageName[len(imageName)-1]))
			file, err := os.Create("./mangas/" + uuidManga + "/" + filename)
			if err != nil {
				message := fmt.Sprintf("Erro ao criar o arquivo: %s", err.Error())
				m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
				return nil, err
			}
			defer file.Close()

			response, err := http.Get(imageURL)
			if err != nil {
				message := fmt.Sprintf("Erro ao realizar o download da imagem: %s", err.Error())
				m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
				return nil, err
			}
			defer response.Body.Close()

			_, err = io.Copy(file, response.Body)
			if err != nil {
				message := fmt.Sprintf("Erro ao copiar o conteúdo da imagem para o arquivo:%s", err.Error())
				m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
				return nil, err
			}
		}

		// Clica na div "page-next" para baixar a próxima imagem
		err = chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitVisible("div.page-next", chromedp.ByQuery),
			chromedp.Click("div.page-next", chromedp.ByQuery),
			chromedp.Location(&url),
			chromedp.InnerHTML("html", &html),
		})
		if err != nil {
			message := fmt.Sprintf("Erro ao clicar na div 'page-next': %s", err.Error())
			m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
			return nil, err
		}

		// Verifica se é mesma imagem
		document, err = goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			message := fmt.Sprintf("Erro ao parsear o HTML da página: %s", err.Error())
			m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
			return nil, err
		}
		imageNextURL, _ := document.Find("div.reader-content div.manga-image img").First().Attr("src")

		if imageURL == imageNextURL {
			break
		}
	}

	c <- true

	return nil, err
}

func (m *MangaHandler) GerarPDF(uuidManga string) (*gofpdf.Fpdf, error) {
	m.uLogger.LogIt("DEBUG", "Iniciado Gerar PDF ID: "+uuidManga, nil)
	// Cria um novo PDF com orientação "P" (retrato)
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Obtém a lista de arquivos JPG no diretório
	files, err := ioutil.ReadDir("./mangas/" + uuidManga)
	if err != nil {
		message := fmt.Sprintf("Erro ao encontrar ler diretório: %s", err.Error())
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar PDF ID: %s Error: %s", uuidManga, message), nil)
		return pdf, err
	}

	type file struct {
		name     string
		order    int
		sequence int
	}

	// Criar uma estrutura de dados para armazenar o nome do arquivo, o número antes do ponto e o número após o ponto
	fileList := make([]file, len(files))

	// Percorrer a lista de arquivos e extrair o número antes e após o ponto
	for i, f := range files {
		// Separar o nome do arquivo pelo ponto
		nameSplit := strings.Split(f.Name(), ".")

		// Extrair o número antes do ponto
		order := 0
		if len(nameSplit) > 1 {
			order, _ = strconv.Atoi(nameSplit[0])
		}

		// Extrair o número após o ponto
		sequence := 0
		if len(nameSplit) > 2 {
			sequence, _ = strconv.Atoi(nameSplit[1])
		}

		// Adicionar o nome do arquivo, o número antes do ponto e o número após o ponto na estrutura de dados
		fileList[i] = file{name: f.Name(), order: order, sequence: sequence}
	}

	// Ordenar a estrutura de dados baseada no número antes do ponto e, em seguida, no número após o ponto
	sort.Slice(fileList, func(i, j int) bool {
		if fileList[i].order == fileList[j].order {
			return fileList[i].sequence < fileList[j].sequence
		}
		return fileList[i].order < fileList[j].order
	})

	// Percorre todos os arquivos e adiciona cada imagem ao PDF
	for _, file := range fileList {
		if filepath.Ext(file.name) == ".jpg" || filepath.Ext(file.name) == ".png" {
			m.addImageToPDF(pdf, "./mangas/"+uuidManga+"/"+file.name, strings.Replace(filepath.Ext(file.name), ".", "", 1))
		}
	}

	/* // Salva o PDF em um arquivo
	err = pdf.OutputFileAndClose("./mangas/" + uuidManga + "/" + "manga.pdf")
	if err != nil {
		message := fmt.Sprintf("Erro ao salvar pdf no diretório: %s", err.Error())
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar PDF ID: %s Error: %s", uuidManga, message), nil)
		return pdf, err
	} */

	return pdf, nil
}

func (m *MangaHandler) addImageToPDF(pdf *gofpdf.Fpdf, imagePath string, imageType string) error {

	m.uLogger.LogIt("DEBUG", fmt.Sprintf("Iniciado addImageToPDF imagePath: %s ", imagePath), nil)

	if imageType == "jpg" {
		fileCheck, err := os.Open(imagePath)
		if err != nil {
			message := fmt.Sprintf("Erro ao abrir imagem no diretório: %s", err.Error())
			m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF imagePath: %s Error: %s", imagePath, message), nil)
			return err
		}
		defer fileCheck.Close()

		// Decode the image file using "go-libjpeg/jpeg"
		_, err = jpeg.Decode(fileCheck, &jpeg.DecoderOptions{})
		if err != nil {
			m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF jpeg.Decode: %s Error: %s", imagePath, err.Error()), nil)
			return nil
		}

		err = fileCheck.Close()
		if err != nil {
			message := fmt.Sprintf("Erro ao fechar imagem no diretório: %s", err.Error())
			m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF imagePath: %s Error: %s", imagePath, message), nil)
			return err
		}
	}

	if imageType == "png" {
		fileCheck, err := os.Open(imagePath)
		if err != nil {
			message := fmt.Sprintf("Erro ao abrir imagem no diretório: %s", err.Error())
			m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF imagePath: %s Error: %s", imagePath, message), nil)
			return err
		}
		defer fileCheck.Close()

		_, err = png.Decode(fileCheck)
		if err != nil {
			m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF png.Decode: %s Error: %s", imagePath, err.Error()), nil)
			return nil
		}

		err = fileCheck.Close()
		if err != nil {
			message := fmt.Sprintf("Erro ao fechar imagem no diretório: %s", err.Error())
			m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF imagePath: %s Error: %s", imagePath, message), nil)
			return err
		}
	}

	// Abre o arquivo de imagem
	file, err := os.Open(imagePath)
	if err != nil {
		message := fmt.Sprintf("Erro ao abrir imagem no diretório: %s", err.Error())
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF imagePath: %s Error: %s", imagePath, message), nil)
		return err
	}
	defer file.Close()

	pdf.AddPage()

	// Converte a imagem para um objeto de imagem PDF
	imgRect := gofpdf.ImageOptions{
		ImageType:             imageType,
		AllowNegativePosition: true,
	}

	w, h := pdf.GetPageSize()
	pdf.RegisterImageOptionsReader(imagePath, imgRect, file)
	pdf.ImageOptions(imagePath, 0, 0, w, h, false, gofpdf.ImageOptions{}, 0, "")

	err = file.Close()
	if err != nil {
		message := fmt.Sprintf("Erro ao fechar imagem no diretório: %s", err.Error())
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF imagePath: %s Error: %s", imagePath, message), nil)
		return err
	}
	return nil
}

func (m *MangaHandler) BuscaListaCapitulos(id string, capInicio string, capFim string) ([]string, error) {
	utils.NewGenericLogger()

	m.uLogger.LogIt("DEBUG", "Iniciado BuscaListaCapitulos", nil)

	var html string
	var uris []string
	// Define o contexto
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Define o timeout para a execução das ações
	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	url := m.vars.MANGA_URL + "/manga/" + id

	// Visita a página desejada
	err := chromedp.Run(ctx, chromedp.Navigate(url))
	if err != nil {
		message := fmt.Sprintf("Erro ao iniciar chromedp: %s", err.Error())
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao BuscaListaCapitulos Error: %s", message), nil)
		return uris, err
	}

	for {
		err := chromedp.Run(ctx,
			// Option 1 to scroll the page: window.scrollTo.
			chromedp.WaitVisible("ul.full-chapters-list", chromedp.ByQuery),
			chromedp.Evaluate(`window.scrollTo(0, document.documentElement.scrollHeight)`, nil),
			// Slow down the action so we can see what happen.
			chromedp.Sleep(1*time.Millisecond),
			// Option 2 to scroll the page: send "End" key to the page.
			chromedp.KeyEvent(kb.End),
			chromedp.Sleep(1*time.Millisecond),
			chromedp.InnerHTML("ul.full-chapters-list", &html),
		)

		if err != nil {
			message := fmt.Sprintf("Erro ao rolar página em busca de capítulos: %s", err.Error())
			m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao BuscaListaCapitulos  Error: %s", message), nil)
			return uris, err
		}

		document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			message := fmt.Sprintf("Erro ao parsear o HTML da página em busca de capítulos: %s", err.Error())
			m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao BuscaListaCapitulos Error: %s", message), nil)
			return uris, err
		}
		title, _ := document.Find(fmt.Sprintf("li a[title='Ler Capítulo %s']", capInicio)).First().Attr("title")

		if title == fmt.Sprintf("Ler Capítulo %s", capInicio) {
			break
		}

		if title == "Ler Capítulo 1" {
			break
		}
	}

	document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		message := fmt.Sprintf("Erro ao parsear o HTML da página em busca de uris: %s", err.Error())
		m.uLogger.LogIt("ERROR", fmt.Sprintf("Erro ao BuscaListaCapitulos Error: %s", message), nil)
		return uris, err
	}

	links := document.Find(`li`)
	for i := range links.Nodes {
		single := links.Eq(i)
		uri, existe := single.Find("a").Attr("href")
		if existe {
			lastIndex := strings.LastIndex(uri, "/")
			cap := uri[lastIndex+1:]
			capNum, _ := strconv.ParseFloat(cap, 32)
			capNumFim, _ := strconv.ParseFloat(capFim, 32)
			capNumInicio, _ := strconv.ParseFloat(capInicio, 32)

			if capNum <= capNumFim && capNum >= capNumInicio {
				uris = append(uris, uri)
			}
		}
	}

	for i, j := 0, len(uris)-1; i < j; i, j = i+1, j-1 {
		uris[i], uris[j] = uris[j], uris[i]
	}

	return uris, nil
}

func (m *MangaHandler) TestaConexao() error {
	m.uLogger.LogIt("DEBUG", fmt.Sprintf("Teste Conexão URL: %s", m.vars.MANGA_URL), nil)
	url := m.vars.MANGA_URL

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("Erro ao criar request URL: %s Error: %s", url, err.Error()))
	}
	req.Header.Set("Accept", `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7`)
	req.Header.Set("Accept-Encoding", `gzip, deflate, br`)
	req.Header.Set("Accept-Language", `pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7`)
	req.Header.Set("sec-ch-ua", `"Chromium";v="110", "Not A(Brand";v="24", "Google Chrome";v="110"`)
	req.Header.Set("sec-ch-ua-mobile", `?0`)
	req.Header.Set("sec-ch-ua-platform", `"Linux"`)
	req.Header.Set("Sec-Fetch-Dest", `document`)
	req.Header.Set("Sec-Fetch-Mode", `navigate`)
	req.Header.Set("Sec-Fetch-Site", `same-origin`)
	req.Header.Set("Sec-Fetch-User", `?1`)
	req.Header.Set("Upgrade-Insecure-Requests", `1`)
	req.Header.Set("User-Agent", `Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36`)

	res, err := client.Do(req)
	if err != nil {
		return errors.New(fmt.Sprintf("Erro ao request URL: %s Error: %s", url, err.Error()))
	}

	if res.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Erro ao acessar a URL: %s\n", res.Status))
	}

	m.uLogger.LogIt("DEBUG", fmt.Sprintf("Teste Conexão URL: %s Status: %s", url, res.Status), nil)

	return nil
}
