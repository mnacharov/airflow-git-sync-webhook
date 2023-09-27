FROM golang:1.21.1 as builder

WORKDIR /build/airflow-git-sync-webhook

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /app/airflow-git-sync-webhook /build/airflow-git-sync-webhook

FROM debian:12.1

COPY --from=builder /app/airflow-git-sync-webhook /app/

ENTRYPOINT ["/app/airflow-git-sync-webhook"]
