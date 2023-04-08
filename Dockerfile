RUN wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | 
apt-key add - \
&& echo "deb http://dl.google.com/linux/chrome/deb/ stable main" >> 
/etc/apt/sources.list.d/google.list
RUN apt-get update && apt-get -y install google-chrome-stable
RUN chrome &
WORKDIR /app/svc/worker
RUN go build -o main .
EXPOSE 6061
CMD ["./main"]




wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | sudo apt-key add -
-----
sudo sh -c 'echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list'

sudo apt-get update 
sudo apt-get install google-chrome-stable