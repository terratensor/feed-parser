# syntax = docker/dockerfile:1.2
FROM golang:latest AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

ARG INDEX_NOW_KEY
ENV INDEX_NOW_KEY=${INDEX_NOW_KEY}

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o ./feed-parser-service ./cmd/service

# 2

FROM scratch

WORKDIR /app

COPY --from=builder /app/feed-parser-service /app/feed-parser-service
COPY --from=builder /app/config /app/config
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Корневой сертификат удостоверяющего центра (УЦ) Минцифры
COPY --from=builder /app/certs/ /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Europe/Moscow

EXPOSE 8080

CMD ["./feed-parser-service"]
