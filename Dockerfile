# Use an official Golang runtime as a parent image
FROM golang:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files into the container
COPY go.mod go.sum ./

# Download and install dependencies
RUN go mod download

# Copy the rest of the application source code into the container
COPY . .

# Build the Go application
RUN go build -o proxifier-app

# Expose the port your application listens on
EXPOSE 8080

# Run your application when the container starts
CMD ["./proxifier-app"]
