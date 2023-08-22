# Start from the base Go image
FROM golang:1.20-alpine as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the source from the current directory to the Working Directory inside the container
COPY electricitymap_exporter.go .
COPY go.mod .
COPY go.sum .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o electricitymap_exporter .

######## Start a new stage from scratch #######
FROM alpine:latest  

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/electricitymap_exporter .

# Expose port 8000 to the outside world
EXPOSE 8000

# Command to run the executable
CMD ["./electricitymap_exporter"]
