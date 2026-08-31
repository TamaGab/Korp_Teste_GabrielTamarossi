FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /usr/local/bin/billing-service ./cmd/api

EXPOSE 8082

CMD ["billing-service"]
