# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git python3 py3-pip gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN python3 -m venv /opt/ocr-venv && \
    /opt/ocr-venv/bin/pip install --upgrade pip && \
    /opt/ocr-venv/bin/pip install -r tools/ocr_parser/requirements.txt && \
    chmod +x tools/ocr_parser/parser.py

RUN CGO_ENABLED=1 GOOS=linux go build -o server cmd/server/main.go

# Stage 3: Production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata python3 sqlite

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /app/server .
COPY --from=builder /opt/ocr-venv /opt/ocr-venv
COPY --from=builder /app/tools/ocr_parser tools/ocr_parser

COPY --from=frontend-builder /app/web/dist web/dist
COPY --from=builder /app/web/static web/static
COPY --from=builder /app/web/templates web/templates
COPY --chown=appuser:appgroup migrations/ migrations/
COPY --chown=appuser:appgroup keys/ keys/

RUN mkdir -p uploads logs archives && \
    chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./server"]
