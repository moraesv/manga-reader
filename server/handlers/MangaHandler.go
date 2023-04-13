package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
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

func NewManga(e *mux.Router) {
	s := e.PathPrefix("/").Subrouter()
	s.HandleFunc("/manga", DownloadManga).Methods("POST")
}

func response(writer http.ResponseWriter, statusCode int, message string) {
	writer.WriteHeader(500)
	resp := make(map[string]string)
	resp["message"] = message
	jsonResp, _ := json.Marshal(resp)
	writer.Write(jsonResp)
}

func DownloadManga(writer http.ResponseWriter, request *http.Request) {
	utilsLogger := utils.NewGenericLogger()

	uuidManga := uuid.New().String()

	utilsLogger.LogIt("DEBUG", "Iniciado DownloadManga ID: "+uuidManga, nil)

	// Decodifica o corpo da requisição
	var body models.BodyManga
	err := json.NewDecoder(request.Body).Decode(&body)
	if err != nil {
		response(writer, http.StatusBadRequest, fmt.Sprintf("Erro ao decodificar JSON: %s", err.Error()))
		return
	}

	err = TestaConexao()
	if err != nil {
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao TestaConexao ID: %s Error: %s", uuidManga, err.Error()), nil)
		response(writer, http.StatusInternalServerError, err.Error())
		return
	}

	uris, err := BuscaListaCapitulos(body.ID, body.CapInicio, body.CapFim)
	if err != nil {
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao BuscaListaCapitulos ID: %s Error: %s", uuidManga, err.Error()), nil)
		response(writer, http.StatusInternalServerError, err.Error())
		return
	}

	err = os.MkdirAll("./mangas/"+uuidManga, os.ModePerm)
	if err != nil {
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao criar diretório ID: %s Error: %s", uuidManga, err.Error()), nil)
		response(writer, http.StatusInternalServerError, err.Error())
		return
	}

	err = GerarManga(uuidManga, uris)
	if err != nil {
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao GerarManga ID: %s Error: %s", uuidManga, err.Error()), nil)
		response(writer, http.StatusInternalServerError, err.Error())
		return
	}

	http.ServeFile(writer, request, "./mangas/"+uuidManga+"/manga.pdf")

	utilsLogger.LogIt("DEBUG", "PDF enviado", nil)

	err = os.RemoveAll("./mangas/" + uuidManga)
	if err != nil {
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao excluir diretório ID: %s Error: %s", uuidManga, err.Error()), nil)
		return
	}
}

func GerarManga(uuidManga string, uris []string) error {
	utilsLogger := utils.NewGenericLogger()

	utilsLogger.LogIt("DEBUG", "Iniciado Gerar Manga ID: "+uuidManga, nil)

	count := 0

	for _, uri := range uris {
		utilsLogger.LogIt("DEBUG", fmt.Sprintf("Iniciado Gerar Manga ID: %s Capítulo %s", uuidManga, uri), nil)

		// URL da página web que queremos acessar
		url := os.Getenv("MANGA_URL") + uri

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
			utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
			return err
		}

		// Clica na div "page-next" e baixa a próxima imagem até que não exista mais uma imagem na tela
		for {
			// Faz o download da imagem
			document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
			if err != nil {
				message := fmt.Sprintf("Erro ao parsear o HTML da página: %s", err.Error())
				utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
				return err
			}
			imageURL, _ := document.Find("div.reader-content div.manga-image img").First().Attr("src")

			if imageURL != "" {
				count += 1
				imageName := strings.Split(imageURL, "/")
				filename := fmt.Sprintf("%d%s", count, filepath.Ext(imageName[len(imageName)-1]))
				file, err := os.Create("./mangas/" + uuidManga + "/" + filename)
				if err != nil {
					message := fmt.Sprintf("Erro ao criar o arquivo: %s", err.Error())
					utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
					return err
				}
				defer file.Close()

				response, err := http.Get(imageURL)
				if err != nil {
					message := fmt.Sprintf("Erro ao realizar o download da imagem: %s", err.Error())
					utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
					return err
				}
				defer response.Body.Close()

				_, err = io.Copy(file, response.Body)
				if err != nil {
					message := fmt.Sprintf("Erro ao copiar o conteúdo da imagem para o arquivo:%s", err.Error())
					utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
					return err
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
				utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
				return err
			}

			// Verifica se é mesma imagem
			document, err = goquery.NewDocumentFromReader(strings.NewReader(html))
			if err != nil {
				message := fmt.Sprintf("Erro ao parsear o HTML da página: %s", err.Error())
				utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar Manga ID: %s Error: %s", uuidManga, message), nil)
				return err
			}
			imageNextURL, _ := document.Find("div.reader-content div.manga-image img").First().Attr("src")

			if imageURL == imageNextURL {
				break
			}
		}
	}

	err := GerarPDF(uuidManga)

	return err
}

func GerarPDF(uuidManga string) error {
	utilsLogger := utils.NewGenericLogger()

	utilsLogger.LogIt("DEBUG", "Iniciado Gerar PDF ID: "+uuidManga, nil)
	// Cria um novo PDF com orientação "P" (retrato)
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Obtém a lista de arquivos JPG no diretório
	files, err := ioutil.ReadDir("./mangas/" + uuidManga)
	if err != nil {
		message := fmt.Sprintf("Erro ao encontrar ler diretório: %s", err.Error())
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar PDF ID: %s Error: %s", uuidManga, message), nil)
		return err
	}

	// Ordena os arquivos por ordem numérica crescente
	sort.Slice(files, func(i, j int) bool {
		fileA := files[i]
		fileB := files[j]
		baseA := filepath.Base(fileA.Name())
		baseB := filepath.Base(fileB.Name())
		extA := filepath.Ext(baseA)
		extB := filepath.Ext(baseB)
		numA, _ := strconv.Atoi(strings.TrimSuffix(baseA, extA))
		numB, _ := strconv.Atoi(strings.TrimSuffix(baseB, extB))
		return numA < numB
	})

	// Percorre todos os arquivos e adiciona cada imagem ao PDF
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".jpg" || filepath.Ext(file.Name()) == ".png" {
			addImageToPDF(pdf, "./mangas/"+uuidManga+"/"+file.Name(), strings.Replace(filepath.Ext(file.Name()), ".", "", 1))
		}
	}

	// Salva o PDF em um arquivo
	err = pdf.OutputFileAndClose("./mangas/" + uuidManga + "/" + "manga.pdf")
	if err != nil {
		message := fmt.Sprintf("Erro ao salvar pdf no diretório: %s", err.Error())
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao Gerar PDF ID: %s Error: %s", uuidManga, message), nil)
		return err
	}

	return nil
}

func addImageToPDF(pdf *gofpdf.Fpdf, imagePath string, imageType string) error {
	utilsLogger := utils.NewGenericLogger()

	utilsLogger.LogIt("DEBUG", fmt.Sprintf("Iniciado addImageToPDF imagePath: %s ", imagePath), nil)

	if imageType == "jpg" {
		fileCheck, err := os.Open(imagePath)
		if err != nil {
			message := fmt.Sprintf("Erro ao abrir imagem no diretório: %s", err.Error())
			utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF imagePath: %s Error: %s", imagePath, message), nil)
			return err
		}
		defer fileCheck.Close()

		// Decode the image file using "go-libjpeg/jpeg"
		_, err = jpeg.Decode(fileCheck, &jpeg.DecoderOptions{
			ScaleTarget: image.Rect(0, 0, 800, 800),
		})
		if err != nil {
			utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF jpeg.Decode: %s Error: %s", imagePath, err.Error()), nil)
			return nil
		}

		err = fileCheck.Close()
		if err != nil {
			message := fmt.Sprintf("Erro ao fechar imagem no diretório: %s", err.Error())
			utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF imagePath: %s Error: %s", imagePath, message), nil)
			return err
		}
	}

	// Abre o arquivo de imagem
	file, err := os.Open(imagePath)
	if err != nil {
		message := fmt.Sprintf("Erro ao abrir imagem no diretório: %s", err.Error())
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF imagePath: %s Error: %s", imagePath, message), nil)
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
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao addImageToPDF imagePath: %s Error: %s", imagePath, message), nil)
		return err
	}
	return nil
}

func BuscaListaCapitulos(id string, capInicio string, capFim string) ([]string, error) {
	utilsLogger := utils.NewGenericLogger()

	utilsLogger.LogIt("DEBUG", "Iniciado BuscaListaCapitulos", nil)

	var html string
	var uris []string
	// Define o contexto
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Define o timeout para a execução das ações
	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	url := os.Getenv("MANGA_URL") + "/manga/" + id

	// Visita a página desejada
	err := chromedp.Run(ctx, chromedp.Navigate(url))
	if err != nil {
		message := fmt.Sprintf("Erro ao iniciar chromedp: %s", err.Error())
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao BuscaListaCapitulos Error: %s", message), nil)
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
			utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao BuscaListaCapitulos  Error: %s", message), nil)
			return uris, err
		}

		document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			message := fmt.Sprintf("Erro ao parsear o HTML da página em busca de capítulos: %s", err.Error())
			utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao BuscaListaCapitulos Error: %s", message), nil)
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
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao BuscaListaCapitulos Error: %s", message), nil)
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

func TestaConexao() error {
	utilsLogger := utils.NewGenericLogger()

	url := os.Getenv("MANGA_URL")
	var res *http.Response

	res, err := http.Head(url)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Erro ao acessar a URL: %s\n", res.Status))
	}

	utilsLogger.LogIt("DEBUG", fmt.Sprintf("Teste Conexão URL: %s Status: %s", url, res.Status), nil)

	return nil
}
