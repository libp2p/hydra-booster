FROM golang:1.12-stretch

EXPOSE 10000-12000

COPY . /dht-node

RUN cd /dht-node && go build

CMD ["/dht-node/dht-node"]
