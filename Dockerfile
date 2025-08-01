# Start from the official Go image
FROM golang:1.21-alpine

# Install git (for go get) and bash (for scripts)
RUN apk add --no-cache git bash

# Set working directory
WORKDIR /app

# Copy go.mod first for better caching
COPY go.mod .

# Create an empty go.sum if it does not exist, then copy it (if present)
RUN touch go.sum
COPY go.sum .

RUN go mod download || true

# Copy the rest of the source code
COPY . .

# Install golint
RUN go install golang.org/x/lint/golint@latest

# Default command (will fail until main.go exists)
CMD ["go", "run", "./cmd/main.go"]