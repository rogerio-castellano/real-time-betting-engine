# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o backend .

# Stage 2: Minimal final image
FROM alpine:3.20

# Copy the executable from builder stage
COPY --from=builder /app/backend /backend

# Set binary as the entrypoint
EXPOSE 8082
ENTRYPOINT ["/backend"]

