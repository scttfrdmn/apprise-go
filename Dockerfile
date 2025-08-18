# Multi-stage Docker build for Apprise-Go
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o apprise-cli cmd/apprise/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o apprise-api cmd/apprise-api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o apprise-docs cmd/apprise-docs/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o apprise-migrate cmd/apprise-migrate/main.go

# Final stage - minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create app user and directory
RUN adduser -D -s /bin/sh apprise
WORKDIR /home/apprise

# Copy binaries from builder
COPY --from=builder /app/apprise-cli ./bin/apprise-cli
COPY --from=builder /app/apprise-api ./bin/apprise-api
COPY --from=builder /app/apprise-docs ./bin/apprise-docs
COPY --from=builder /app/apprise-migrate ./bin/apprise-migrate

# Copy configuration templates
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/docs ./docs

# Set permissions
RUN chown -R apprise:apprise /home/apprise
RUN chmod +x /home/apprise/bin/*

# Switch to app user
USER apprise

# Expose API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command runs the API server
CMD ["./bin/apprise-api"]