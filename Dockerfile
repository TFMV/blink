# syntax=docker/dockerfile:1

############################
# Build Stage
############################
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Cache dependencies by copying go.mod and go.sum separately
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary with optimizations for size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o blink ./cmd/blink

############################
# Final Stage
############################
# Use a minimal, secure distroless image with a non-root user
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

# Copy the statically compiled binary from the builder stage
COPY --from=builder /app/blink .

EXPOSE 12345

ENTRYPOINT ["/app/blink"]
CMD ["--event-addr", ":12345", "--path", "/watch"]
