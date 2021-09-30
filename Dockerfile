# syntax=docker/dockerfile:1

FROM golang:alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

COPY *.html ./

COPY static ./static

RUN go build -o /voting-server

EXPOSE 8080

CMD [ "/voting-server" ]
