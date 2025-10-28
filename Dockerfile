# --- Build stage ---
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Install git for go mod download if needed
RUN apk add --no-cache git

# Copy go mod/sum and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go binary (static build)
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -o fleet-backend ./cmd/main.go

# --- Final stage (minimal) ---
FROM alpine:3.20

WORKDIR /app

# Copy only the built binary
COPY --from=builder /app/fleet-backend .

# Expose the default port
EXPOSE 8080

# Run the binary
CMD ["./fleet-backend"]