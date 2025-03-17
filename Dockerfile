# 🏗️ Stage 1: Build the Go Binary
FROM golang:1.23 AS builder

# Set working directory inside the container
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the Go binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o watch-tower main.go

# 🏗️ Stage 2: Create a Minimal Runtime Image
FROM alpine:latest

# Set working directory
WORKDIR /root/

# Install certificates (needed for HTTPS Kubernetes API calls)
RUN apk add --no-cache ca-certificates

# Copy the compiled binary from builder stage
COPY --from=builder /app/watch-tower /usr/local/bin/watch-tower

# Ensure the binary has executable permissions
RUN chmod +x /usr/local/bin/watch-tower

# Run as a non-root user for security
RUN addgroup -S watchtower && adduser -S watchtower -G watchtower
USER watchtower

# Default environment variables (override at runtime)
ENV DB_CREDENTIAL_PATH="/srv/db_credential"
ENV AAP_NAMESPACE="default"

# Entrypoint for the controller
ENTRYPOINT ["/usr/local/bin/watch-tower"]

