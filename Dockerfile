# Dockerfile
FROM golang:1.23-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY . .

RUN /go/bin/swag init

RUN go build -o /bin/main .

CMD ["/bin/main"]