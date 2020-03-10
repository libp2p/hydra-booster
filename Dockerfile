FROM golang:1.14-stretch

WORKDIR /hydra-booster

COPY go.mod go.sum ./
RUN go mod download

# Copy the source from the current directory 
# to the Working Directory inside the container
COPY . .

RUN go build -o hydra-booster .

EXPOSE 10000-12000
RUN ls
CMD ["./hydra-booster -many=50 -portBegin=10000"]
