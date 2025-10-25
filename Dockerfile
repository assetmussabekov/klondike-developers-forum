# Build stage: Alpine with build tools
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go app (CGO enabled for SQLite)
RUN CGO_ENABLED=1 go build -o server ./cmd/server

# Final stage: minimal Alpine image
FROM alpine:latest

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/server .

# Copy static files
COPY static ./static

# Optionally copy the database if you want to ship with a pre-existing one
# COPY forum.db ./forum.db

EXPOSE 8080

CMD ["./server"]