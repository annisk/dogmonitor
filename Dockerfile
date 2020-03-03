FROM golang:latest

WORKDIR /usr/src/app

RUN go get github.com/slack-go/slack github.com/mattn/go-sqlite3

COPY dog.go .

RUN go build dog.go

ENTRYPOINT ["/usr/src/app/dog"]
