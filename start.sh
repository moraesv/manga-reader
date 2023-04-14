#!/bin/bash

# Nome da imagem Docker
DOCKER_IMAGE_NAME="manga-reader"
PORTA=9003
DIR_PATH=/home/vinicius/projetos/go/src/github.com/pessoal/manga-reader

echo "Removendo container Docker..."
docker stop $DOCKER_IMAGE_NAME && docker rm $DOCKER_IMAGE_NAME

# Construindo a imagem Docker
echo "Construindo a imagem Docker..."
docker build -f $DIR_PATH/Dockerfile -t $DOCKER_IMAGE_NAME $DIR_PATH

# Iniciando o container
echo "Iniciando o container Docker..."
docker run --env-file $DIR_PATH/.env -p $PORTA:$PORTA --name $DOCKER_IMAGE_NAME $DOCKER_IMAGE_NAME
