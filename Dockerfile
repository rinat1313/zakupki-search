# syntax=docker/dockerfile:1
FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/search ./cmd/search

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
  && rm -rf /var/lib/apt/lists/*
# Сертификаты Минцифры (с 2026 zakupki.gov.ru на Russian Trusted CA).
COPY certs/*.crt /usr/local/share/ca-certificates/
RUN update-ca-certificates
WORKDIR /app
COPY --from=build /out/search /usr/local/bin/search
COPY migrations /app/migrations
COPY certs /app/certs
ENV HTTP_ADDR=:8091
ENV MIGRATIONS_DIR=/app/migrations
ENV EIS_CA_DIR=/app/certs
EXPOSE 8091
HEALTHCHECK CMD curl -fsS http://127.0.0.1:8091/health || exit 1
CMD ["search"]
