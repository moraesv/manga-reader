FROM golang:1.19
RUN wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add - 
RUN echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list
RUN apt-get update
RUN apt-get -y install google-chrome-stable
RUN apt-get -y install libjpeg-dev
RUN chrome &
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go build -o main .
EXPOSE 5001
CMD ["/app/main"]
