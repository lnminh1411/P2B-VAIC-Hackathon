FROM golang:1.26-bookworm AS api-builder
WORKDIR /src/api
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
ARG TARGET=api
RUN CGO_ENABLED=0 go build -o /out/service ./cmd/${TARGET}

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates poppler-utils tesseract-ocr tesseract-ocr-vie libreoffice-writer fonts-noto-core clamav \
    && rm -rf /var/lib/apt/lists/*
RUN useradd --create-home --uid 10001 p2b
COPY --from=api-builder /out/service /usr/local/bin/p2b-service
USER p2b
ENV HTTP_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/p2b-service"]

