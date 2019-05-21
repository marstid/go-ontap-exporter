
# Start from golang v1.12 base image
FROM golang:1.12 as builder

# Add Maintainer Info
LABEL maintainer="MS <marstid@juppi.net>"

# Set the Current Working Directory inside the container
WORKDIR /go/src/github.com/marstid/go-ontap-exporter

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Download dependencies
RUN go get -d -v ./...

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/go-netapp .


######## Start a new stage from scratch #######
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /go/bin/go-netapp .

EXPOSE 9099

CMD ["./go-netapp"]