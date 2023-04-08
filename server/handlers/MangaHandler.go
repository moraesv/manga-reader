package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"manga-reader/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jung-kurt/gofpdf"
)

func NewManga(e *mux.Router) {
	s := e.PathPrefix("/").Subrouter()
	s.HandleFunc("/manga", DownloadManga).Methods("GET")
}

func DownloadManga(writer http.ResponseWriter, request *http.Request) {
	utilsLogger := utils.NewGenericLogger()

	uuidManga := uuid.New().String()

	utilsLogger.LogIt("DEBUG", "Iniciado DownloadManga ID: "+uuidManga, nil)

	buscalista()

	json.NewEncoder(writer).Encode(nil)
	return

	err := os.Mkdir(uuidManga, 0755)
	if err != nil {
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao criar diretório ID: %s Error: %s", uuidManga, err.Error()), nil)
		return
	}

	gerarManga(uuidManga)

	http.ServeFile(writer, request, "./"+uuidManga+"/manga.pdf")

	utilsLogger.LogIt("DEBUG", "PDF enviado", nil)

	err = os.RemoveAll("./" + uuidManga)
	if err != nil {
		utilsLogger.LogIt("ERROR", fmt.Sprintf("Erro ao excluir diretório ID: %s Error: %s", uuidManga, err.Error()), nil)
		return
	}
}

func gerarManga(uuidManga string) {
	utilsLogger := utils.NewGenericLogger()

	utilsLogger.LogIt("DEBUG", "Iniciado Gerar Manga ID: "+uuidManga, nil)

	// URL da página web que queremos acessar
	url := "https://mangalivre.net/ler/boku-no-hero-academia/online/454062/38#/!page0"

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
		fmt.Println("Erro ao navegar até a página web:", err)
		os.Exit(1)
	}

	// Clica na div "page-next" e baixa a próxima imagem até que não exista mais uma imagem na tela
	for {
		// Faz o download da imagem
		document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			fmt.Println("Erro ao parsear o HTML da página:", err)
			os.Exit(1)
		}
		imageURL, _ := document.Find("div.reader-content div.manga-image img").First().Attr("src")

		if imageURL != "" {
			imageName := strings.Split(imageURL, "/")
			file, err := os.Create(uuidManga + "/" + imageName[len(imageName)-1])
			if err != nil {
				fmt.Println("Erro ao criar o arquivo:", err)
				os.Exit(1)
			}
			defer file.Close()

			response, err := http.Get(imageURL)
			if err != nil {
				fmt.Println("Erro ao realizar o download da imagem:", err)
				os.Exit(1)
			}
			defer response.Body.Close()

			_, err = io.Copy(file, response.Body)
			if err != nil {
				fmt.Println("Erro ao copiar o conteúdo da imagem para o arquivo:", err)
				os.Exit(1)
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
			fmt.Println("Erro ao clicar na div 'page-next':", err)
			os.Exit(1)
		}

		// Verifica se é mesma imagem
		document, err = goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			fmt.Println("Erro ao parsear o HTML da página:", err)
			os.Exit(1)
		}
		imageNextURL, _ := document.Find("div.reader-content div.manga-image img").First().Attr("src")

		if imageURL == imageNextURL {
			break
		}
	}

	geraPDF(uuidManga)
}

func geraPDF(uuidManga string) {

	// Cria um novo PDF com orientação "P" (retrato)
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Obtém a lista de arquivos JPG no diretório
	files, err := ioutil.ReadDir("./" + uuidManga)
	if err != nil {
		panic(err)
	}

	// Percorre todos os arquivos e adiciona cada imagem ao PDF
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".jpg" || filepath.Ext(file.Name()) == ".png" {
			pdf.AddPage()
			addImageToPDF(pdf, "./"+uuidManga+"/"+file.Name(), strings.Replace(filepath.Ext(file.Name()), ".", "", 1))
		}
		os.Remove("./" + uuidManga + "/" + file.Name())
	}

	// Salva o PDF em um arquivo
	err = pdf.OutputFileAndClose("./" + uuidManga + "/" + "manga.pdf")
	if err != nil {
		panic(err)
	}
}

func addImageToPDF(pdf *gofpdf.Fpdf, imagePath string, imageType string) {
	// Abre o arquivo de imagem
	file, err := os.Open(imagePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Converte a imagem para um objeto de imagem PDF
	imgRect := gofpdf.ImageOptions{
		ImageType:             imageType,
		AllowNegativePosition: true,
	}

	w, h := pdf.GetPageSize()
	pdf.RegisterImageOptionsReader(imagePath, imgRect, file)
	pdf.ImageOptions(imagePath, 0, 0, w, h, false, gofpdf.ImageOptions{}, 0, "")
	file.Close()
}

func buscalista() {
	var html string
	// Define o contexto
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	/* // Define o timeout para a execução das ações
	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel() */

	// Visita a página desejada
	err := chromedp.Run(ctx, chromedp.Navigate("https://mangalivre.net/manga/one-punch-man/1036"))
	if err != nil {
		log.Fatal(err)
	}

	// Loop para rolar a página até encontrar o elemento desejado
	var nodes []*cdp.Node
	var numItems int
	//var cap1Found bool

	for i := 0; i < 2; i++ {
		if err := chromedp.Run(ctx,
			// Option 1 to scroll the page: window.scrollTo.
			chromedp.WaitVisible("ul.full-chapters-list", chromedp.ByQuery),
			chromedp.Evaluate(`window.scrollTo(0, document.documentElement.scrollHeight)`, nil),
			// Slow down the action so we can see what happen.
			chromedp.Sleep(1*time.Millisecond),
			// Option 2 to scroll the page: send "End" key to the page.
			chromedp.KeyEvent(kb.End),
			chromedp.Sleep(1*time.Millisecond),
			chromedp.InnerHTML("ul.full-chapters-list li", &html),
		); err != nil {
			panic(err)
		}
		fmt.Printf(fmt.Sprintf("%d", i))
		fmt.Println(html)

	}
	fmt.Printf("saiu")

	// Executar as tarefas do chromedp
	err = chromedp.Run(ctx,
		// Contar o número de li na ul de classe full-chapters-list
		chromedp.Nodes("ul.full-chapters-list li", &nodes, chromedp.ByQuery),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("aqui")

	numItems = len(nodes)

	// Verifica se o item desejado está presente
	/* for _, node := range nodes {
		if node.NodeName == "LI" {
			if node.Children[1].Children[0].NodeValue == "Capítulo 1" {
				cap1Found = true
				break
			}
		}
	} */

	/* for !cap1Found {
		// Rola a página até o final
		err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			layoutViewport, _, contentSize, _, _, _, err := page.GetLayoutMetrics().Do(ctx)
			if err != nil {
				return err
			}
			viewportHeight := int64(contentSize.Height)
			windowHeight := layoutViewport.ClientHeight
			if err != nil {
				return err
			}
			for scrollOffset := windowHeight; scrollOffset <= viewportHeight; scrollOffset += windowHeight {
				err := page.Scroll{
					Y: scrollOffset,
				}.Do(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}))
		if err != nil {
			log.Fatal(err)
		}

		// Espera até que os itens sejam carregados
		err = chromedp.Run(ctx, chromedp.Sleep(2*time.Second))
		if err != nil {
			log.Fatal(err)
		}

		// Obtém a lista de elementos li
		err = chromedp.Run(ctx, chromedp.Nodes("//ul[@class='full-chapters-list']/li", &nodes))
		if err != nil {
			log.Fatal(err)
		}
		numItems = len(nodes)

		// Verifica se o item desejado está presente
		for _, node := range nodes {
			if node.NodeName == "LI" {
				if node.Children[1].Children[0].NodeValue == "Capítulo 1" {
					cap1Found = true
					break
				}
			}
		}
	} */

	fmt.Printf("Número de itens na lista: %d\n", numItems)
}
