# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/api-go ./cmd/server

# Final stage
FROM alpine:latest

RUN adduser -D -u 10001 appuser
WORKDIR /home/appuser

# Install CA certificates
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/api-go .
RUN chmod 755 ./api-go
COPY --from=builder /app/docs ./docs

USER 10001

# Expose port
EXPOSE 8080

# Run the application
CMD ["./api-go"]
