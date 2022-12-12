FROM golang:1.18-alpine AS build

RUN apk add --no-cache openssl-dev git build-base

WORKDIR /hydra-booster

COPY go.mod go.sum ./
RUN go mod download -x

# Copy the source from the current directory
# to the Working Directory inside the container
COPY datastore ./datastore
COPY head ./head
COPY httpapi ./httpapi
COPY hydra ./hydra
COPY idgen ./idgen
COPY ui ./ui
COPY utils ./utils
COPY version ./version
COPY metrics ./metrics
COPY metricstasks ./metricstasks
COPY periodictasks ./periodictasks
COPY providers ./providers
COPY testing ./testing
COPY main.go promconfig.yaml ./

# Run the build and install
RUN go install -tags=openssl -v ./...

# Create single-layer run image
FROM alpine@sha256:bc41182d7ef5ffc53a40b044e725193bc10142a1243f395ee852a8d9730fc2ad
RUN apk add --no-cache openssl curl  # curl is for health checking
COPY --from=build /go/bin/hydra-booster /hydra-booster
# HTTP API
COPY entrypoint.sh ./
RUN chmod a+x entrypoint.sh
EXPOSE 7779

# Prometheus /metrics
EXPOSE 8888

# Heads
EXPOSE 30000-32767
EXPOSE 30000-32767/udp

# CMD ["./hydra-booster", "-metrics-addr=0.0.0.0:8888", "-httpapi-addr=0.0.0.0:7779", "-ui-theme=none"]
CMD ["./entrypoint.sh"]
