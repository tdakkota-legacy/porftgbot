FROM golang:latest as builder

MAINTAINER tdakkota

# Set the Current Working Directory inside the container
WORKDIR /app

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
COPY go.mod go.sum ./
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build
ENV CGO_ENABLED=0
RUN go generate ./... && go fmt ./...
RUN go build -v ./cmd/porftgbot

######## Start a new stage from scratch #######
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/
# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/porftgbot .

ENTRYPOINT ["./porftgbot"]