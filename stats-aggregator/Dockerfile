FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o aggregator .

EXPOSE 8082

CMD ["./aggregator"]
