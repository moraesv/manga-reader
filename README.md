# manga-reader

Projeto destinado ao download de mangas em PDF buscando do site mangalivre

### Docker

Depois de instalado versão do `Docker` (windows, linux ou mac), irá executar o comando seguinte comando para verificar se o mesmo foi instalado:
`docker --version`

Para gerar a imagem é preciso acessar a pasta raiz do projeto e executar o comando:
`docker build -f Dockerfile -t manga-reader .`

Para executar o container:
`docker run --env-file .env -p 9003:9003 --name manga-reader manga-reader`

Para gerar a imagem do redirect é preciso acessar a pasta raiz do projeto e executar o comando:
`docker build -f DockerfileRedirect -t manga-reader-redirect .`

Para executar o container:
`docker run --env-file .env -p 9004:9004 --name manga-reader-redirect manga-reader-redirect`

### bashrc para start.sh

chmod +x /home/vinicius/projetos/go/src/github.com/pessoal/manga-reader/start.sh
alias manga=". /home/vinicius/projetos/go/src/github.com/pessoal/manga-reader/start.sh"
