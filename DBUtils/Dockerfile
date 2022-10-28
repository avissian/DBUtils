## Swagger spec
FROM quay.io/goswagger/swagger AS goswagger
WORKDIR /data
COPY go.mod ./
COPY go.sum ./
COPY *.go ./
RUN swagger generate spec -o swagger.yml

## Swagger HTML
FROM swaggerapi/swagger-codegen-cli AS swagger-codegen
WORKDIR /data
COPY --from=goswagger /data/swagger.yml /data
RUN java -jar /opt/swagger-codegen-cli/swagger-codegen-cli.jar generate -i swagger.yml -l html2

## Build binary
FROM golang:buster AS builder
WORKDIR /data
COPY go.mod ./
COPY go.sum ./
COPY *.go ./
# ENV http_proxy <HTTP_PROXY>
RUN go build -tags=nomsgpack -ldflags "-s -w" -o /data/dbutils .

## run
FROM ubuntu
WORKDIR /app
COPY --from=builder /data/dbutils ./
COPY --from=swagger-codegen /data/index.html ./static/
COPY ./config.yml ./
CMD /app/dbutils -web
