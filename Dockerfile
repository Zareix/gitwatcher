FROM golang:1.27.1-alpine3.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/gitwatcher ./cmd/gitwatcher


FROM alpine:3.24.1 AS runner

RUN apk add --no-cache git ca-certificates \
    && addgroup -S gitwatcher \
    && adduser -S -G gitwatcher gitwatcher

COPY --from=builder /app/gitwatcher /app/gitwatcher

ENV REPOSITORY_PATH=/repo
ENV CRON="0 * * * * *"
ENV GOMEMLIMIT=32MiB

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD pidof gitwatcher || exit 1

USER gitwatcher

CMD ["/app/gitwatcher"]
