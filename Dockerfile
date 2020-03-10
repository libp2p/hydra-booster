FROM golang:1.14-stretch

EXPOSE 10000-12000

COPY . /hydra-booster

RUN cd /hydra-booster && go build
CMD ["./hydra-booster -many=50 -portBegin=10000"]
