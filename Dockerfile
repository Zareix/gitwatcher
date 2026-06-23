FROM golang:1.26.4-alpine3.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o /app/gitwatcher ./cmd/gitwatcher


FROM alpine:3.24.1 AS runner

RUN apk add --no-cache git ca-certificates

COPY --from=builder /app/gitwatcher /app/gitwatcher

ENV REPOSITORY_PATH=/repo
ENV CRON="* */5 * * * *"
ENV PORT=8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/jobs | grep -q '\[{' || exit 1

CMD ["/app/gitwatcher"]
