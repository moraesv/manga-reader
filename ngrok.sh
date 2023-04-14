#!/bin/bash

# Nome da imagem Docker
DOCKER_IMAGE_NAME="manga-reader"
PORTA=9003
ENV_PATH=/home/vinicius/projetos/go/src/github.com/pessoal/manga-reader/.env

# Verificando se já existe um container com o nome da imagem
if docker ps --format '{{.Names}}' | grep -q "^$DOCKER_IMAGE_NAME$"; then
  echo "Já existe um container com o nome $DOCKER_IMAGE_NAME. Removendo..."
  docker stop $DOCKER_IMAGE_NAME && docker rm $DOCKER_IMAGE_NAME
fi

# Construindo a imagem Docker
# echo "Construindo a imagem Docker..."
# docker build -f Dockerfile -t $DOCKER_IMAGE_NAME .

# Iniciando o container
echo "Iniciando o container Docker..."
docker run -d --env-file $ENV_PATH -p $PORTA:$PORTA --name $DOCKER_IMAGE_NAME $DOCKER_IMAGE_NAME


# Iniciando ngork
# ngrok http $PORTA > /dev/null &

# Obtendo a URL do ngrok
echo "Obtendo a URL do ngrok..."
NGROK_URL=$(curl -s http://localhost:4040/api/tunnels | jq -r '.tunnels[0].public_url')
# Exibindo a URL do ngrok
echo "Acesso público: $NGROK_URL"

# Liberando o terminal
exec bash
