FROM golang:1.13.8-buster

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
# Heads
EXPOSE 30000-32767
CMD ["./hydra-booster", "-metrics-addr=0.0.0.0:8888", "-httpapi-addr=0.0.0.0:7779", "-ui-theme=none"]
