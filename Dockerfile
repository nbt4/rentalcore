# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# Stage 2: Build OCR Python venv (Python 3.12 for pandas compat)
FROM python:3.12-alpine AS ocr-builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY tools/ocr_parser/requirements.txt tools/ocr_parser/

RUN python3 -m venv /opt/ocr-venv && \
    /opt/ocr-venv/bin/pip install --upgrade pip && \
    /opt/ocr-venv/bin/pip install -r tools/ocr_parser/requirements.txt

# Stage 3: Build Backend
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o server cmd/server/main.go

# Stage 4: Production image
# Keep the runtime Python version aligned with the venv built above. A plain
# alpine:latest image can ship a newer /usr/bin/python while the copied venv
# still points at /usr/local/bin/python3 from Python 3.12.
FROM python:3.12-alpine

RUN apk --no-cache add ca-certificates tzdata sqlite

WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /app/server .
COPY --from=ocr-builder /opt/ocr-venv /opt/ocr-venv
COPY --from=builder /app/tools/ocr_parser tools/ocr_parser

# Fail the image build immediately if the OCR runtime or one of its direct
# imports cannot be loaded in the final stage.
RUN /opt/ocr-venv/bin/python3 -c "import click, pandas, pdfplumber, rapidfuzz"

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
