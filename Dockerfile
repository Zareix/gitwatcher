FROM golang:1.25.5-alpine3.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o /app/gitwatcher ./cmd/gitwatcher


FROM gcr.io/distroless/static-debian12 AS runner

COPY --from=builder /app/gitwatcher /app/gitwatcher

ENV REPOSITORY_PATH=/repo
ENV PULLER_JOB_CRON="* */5 * * * *"
ENV PORT=8080

EXPOSE 8080

CMD ["/app/gitwatcher"]
