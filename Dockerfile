# Use the official Golang image
FROM golang:1.24
# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
RUN go build -o ./main ./cmd

# Expose the port the app runs on
EXPOSE 8080

# Run the executable
CMD ["./main"]