# --- Builder Stage ---

# Use the official Golang image as a builder.
FROM golang:1.24-alpine AS builder

# Install git (required for go mod operations with dependencies from git repos).
RUN apk add --no-cache git

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum first for better Docker layer caching.
COPY go.mod go.sum ./

# Download dependencies.
RUN go mod download

# Copy all source code.
COPY . .

# Build the Go application.
# CGO_ENABLED=0 is important for creating a static binary that can run in a minimal image.
# -ldflags "-w -s" strips debugging information, reducing the binary size.
RUN CGO_ENABLED=0 go build -ldflags "-w -s" -o gobds ./cmd/main.go

# --- Final Stage ---

# Use a minimal Alpine image for the final container.
FROM alpine:latest

# Set the working directory.
WORKDIR /app

# Copy the built binary from the builder stage.
COPY --from=builder /app/gobds .

# Copy the example configuration file.
COPY config.example.toml ./config.example.toml

# Create a directory for resources.
RUN mkdir resources

# Create a directory for player data.
# This directory should be mounted as a volume to persist player data.
RUN mkdir players

# Expose the default Bedrock server port.
EXPOSE 19132/udp

# Set the command to run the application.
CMD ["./gobds"]
