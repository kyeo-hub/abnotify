# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod file
COPY go.mod ./

# Copy source code first (needed for go mod tidy)
COPY . .

# Download dependencies
RUN go mod tidy && go mod download

# Build
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o abnotify-server .

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Create data directory
RUN mkdir -p /app/data

# Copy binary
COPY --from=builder /app/abnotify-server .

# Expose port
EXPOSE 8080

# Environment variables
ENV ABNOTIFY_HOST=0.0.0.0
ENV ABNOTIFY_PORT=8080
ENV ABNOTIFY_DB_PATH=/app/data/abnotify.db

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run
CMD ["./abnotify-server"]
""  
