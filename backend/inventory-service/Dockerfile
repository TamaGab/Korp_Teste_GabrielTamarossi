FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /usr/local/bin/inventory-service ./cmd/api

EXPOSE 8081

CMD ["inventory-service"]
