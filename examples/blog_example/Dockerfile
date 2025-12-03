# Multi-stage build for Sargantana Go Blog Example
# Stage 1: Build the Go application
FROM golang:1.25-alpine3.22 AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project (needed to build from examples/blog_example)
COPY . .

# Build the blog example
WORKDIR /build/examples/blog_example
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/blog-server ./cmd/main.go

# Stage 2: Create minimal runtime image
FROM scratch

# Copy CA certificates for HTTPS connections (e.g., to external services)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data (useful for time-based operations)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the compiled binary
COPY --from=builder /app/blog-server /app/blog-server

# Copy static content files (templates, CSS, favicon, etc.)
COPY --from=builder /build/examples/blog_example/content /app/content

# Set working directory
WORKDIR /app

# Expose the application port (matches config.yaml default)
EXPOSE 8080

# Run the blog server
# Note: config.yaml should be mounted as a volume at runtime
# Example: docker run -v $(pwd)/config.yaml:/app/config.yaml ...
ENTRYPOINT ["/app/blog-server"]
CMD ["-config", "/app/config.yaml"]
