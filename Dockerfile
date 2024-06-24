# Use the official Golang image as a build stage
FROM golang:1.22 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -v -o /usr/local/bin/app ./cmd

# Use a more recent base image for the final stage
FROM ubuntu as final

# Copy the compiled Go binary from the builder stage
COPY --from=builder /usr/local/bin/app /usr/local/bin/app

# Command to run the executable
CMD ["/usr/local/bin/app"]