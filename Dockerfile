FROM golang:1.14 AS builder
WORKDIR /usr/src

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN GOOS=linux CGO_ENABLED=0 go build -installsuffix cgo -o search main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /usr/app
COPY --from=0 /usr/src/search .
ENTRYPOINT ["/usr/app/search"]
CMD ["search", "db"]