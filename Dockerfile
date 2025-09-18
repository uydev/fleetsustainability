# --- Build stage ---
FROM golang:1.24.5 AS builder

WORKDIR /app

# Install git and CA certificates for go mod download
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy the source code (including vendor/)
COPY . .

# Build the Go binary using vendored modules
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o fleet-backend ./cmd/main.go

# --- Final stage (minimal) ---
FROM debian:bookworm-slim

WORKDIR /app

# Install CA certificates for TLS at runtime (e.g., HTTPS, Mongo TLS, etc.)
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && update-ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy only the built binary
COPY --from=builder /app/fleet-backend .

# Expose the default port
EXPOSE 8080

# Run the binary
CMD ["./fleet-backend"]