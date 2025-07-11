# Use the official Golang image
FROM golang:1.24

# Set the working directory
WORKDIR /app

COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

COPY backend/main ./main

RUN go build -o main .

EXPOSE 8080

CMD ["./main"]
