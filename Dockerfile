# syntax=docker/dockerfile:1

# --- build stage -----------------------------------------------------------
FROM golang:1.25 AS builder

WORKDIR /src

# Cache modules first for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static build so the binary runs on distroless with no libc.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/crashdummy .

# --- final stage -----------------------------------------------------------
FROM gcr.io/distroless/static:nonroot AS production

WORKDIR /app

# Binary plus the default configuration. A ConfigMap can override these by
# mounting over CRASHDUMMY_CONFIG_DIR (see the k8s manifests in CRASH-008).
COPY --from=builder /out/crashdummy /app/crashdummy
COPY mappings/ /app/mappings/
COPY proxies/ /app/proxies/
COPY stubs/ /app/stubs/

ENV PORT=10000
EXPOSE 10000

USER nonroot:nonroot
ENTRYPOINT ["/app/crashdummy"]
