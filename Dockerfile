FROM golang:1.26.1-alpine3.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o /app/gitwatcher ./cmd/gitwatcher


FROM alpine:3.24.1 AS runner

RUN apk add --no-cache git ca-certificates

COPY --from=builder /app/gitwatcher /app/gitwatcher

ENV REPOSITORY_PATH=/repo
ENV PULLER_JOB_CRON="* */5 * * * *"
ENV PORT=8080

EXPOSE 8080

CMD ["/app/gitwatcher"]
