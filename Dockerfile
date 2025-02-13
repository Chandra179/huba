# Stage 1: Build the Go application
FROM golang:1.24.0-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o app .

# Stage 2: Use Alpine as the final image
FROM alpine:latest

WORKDIR /

# Create a new user and group with limited privileges
RUN addgroup -S appuser && adduser -S appuser -G appuser

# Copy the built binary to /app (this will be a file, not a directory)
COPY --from=builder /app/app /app

# Set ownership of the binary to the new user
RUN chown appuser:appuser /app

# Switch to the new non-root user
USER appuser

EXPOSE 8080

# Run the binary
ENTRYPOINT ["/app"]
