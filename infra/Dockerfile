FROM golang:1.26-bookworm AS api-builder
WORKDIR /src/api
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/p2b-api ./cmd/api \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/p2b-migrate ./cmd/migrate \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/p2b-worker ./cmd/worker \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/p2b-crawler ./cmd/crawler

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates poppler-utils tesseract-ocr tesseract-ocr-vie libreoffice-writer fonts-noto-core clamav \
    && rm -rf /var/lib/apt/lists/*
RUN useradd --create-home --uid 10001 p2b
COPY --from=api-builder /out/p2b-api /usr/local/bin/p2b-service
COPY --from=api-builder /out/p2b-migrate /usr/local/bin/p2b-migrate
COPY --from=api-builder /out/p2b-worker /usr/local/bin/p2b-worker
COPY --from=api-builder /out/p2b-crawler /usr/local/bin/p2b-crawler
USER p2b
ENV HTTP_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/p2b-service"]
