FROM docker.io/library/golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /server ./cmd/server

FROM docker.io/library/alpine:3.21

RUN apk add --no-cache ca-certificates && \
    adduser -D -H -u 10001 appuser

COPY --from=builder --chown=appuser:appuser /server /server

USER appuser

EXPOSE 8080

CMD ["/server"]
