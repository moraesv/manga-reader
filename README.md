# manga-reader

Projeto destinado ao download de mangas em PDF buscando do site mangalivre

### Docker

Depois de instalado versão do `Docker` (windows, linux ou mac), irá executar o comando seguinte comando para verificar se o mesmo foi instalado:
`docker --version`

Para gerar a imagem é preciso acessar a pasta raiz do projeto e executar o comando:
`docker build -f Dockerfile -t manga-reader .`

Para executar o container:
`docker run --env-file .env -p 9003:9003 --name manga-reader manga-reader`
