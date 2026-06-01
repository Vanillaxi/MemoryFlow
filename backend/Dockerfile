FROM golang:1.26-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/memoryflow .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /app/memoryflow-data \
    && ln -s /app/memoryflow-data /memoryflow-data

WORKDIR /app

COPY --from=builder /app/memoryflow /app/memoryflow
COPY configs/config.example.yaml /app/configs/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/memoryflow"]
