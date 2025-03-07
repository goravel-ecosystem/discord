FROM golang:alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0  \
    GOARCH="amd64" \
    GOOS=linux

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:latest

WORKDIR /www

COPY --from=builder /build/main /www/
COPY --from=builder /build/database/ /www/database/
COPY --from=builder /build/public/ /www/public/
COPY --from=builder /build/storage/ /www/storage/
COPY --from=builder /build/resources/ /www/resources/
COPY --from=builder /build/.env /www/.env

ENTRYPOINT ["/www/main"]
