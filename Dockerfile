# Build stage
FROM golang:1.24.3-alpine AS builder

# Install ca-certificates and git for fetching dependencies
RUN apk --no-cache add ca-certificates git

WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire source code
COPY . .

# Build the mandatory-wait plugin specifically
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -a -installsuffix cgo -o mandatory-wait-plugin ./step-plugin-mandatory-wait

# Final stage
FROM alpine:3.18

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/mandatory-wait-plugin /bin/mandatory-wait-plugin

# Make it executable
RUN chmod +x /bin/mandatory-wait-plugin

# Set the entrypoint
ENTRYPOINT ["/bin/mandatory-wait-plugin"]
