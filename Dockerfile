FROM golang:1.14-stretch

WORKDIR /hydra-booster

COPY go.mod go.sum ./
RUN go mod download

# Copy the source from the current directory
# to the Working Directory inside the container
COPY . .

RUN go build -o hydra-booster .

# HTTP API
EXPOSE 7779
# Prometheus /metrics
EXPOSE 8888
# Sybils
EXPOSE 10000-12000
CMD ["./hydra-booster", "-portBegin=10000", "-metrics-port=8888"]
