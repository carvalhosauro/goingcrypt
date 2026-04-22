FROM golang:1.25-alpine AS deps

WORKDIR /app

# Only copy dependency manifests first so this layer is cached until go.mod/go.sum change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

FROM deps AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -trimpath \
      -ldflags="-s -w \
        -X main.version=${VERSION} \
        -X main.commit=${COMMIT} \
        -X main.buildDate=${BUILD_DATE}" \
      -o /out/api \
      ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/migrate \
      ./cmd/migrate

FROM scratch AS migrate

# Copy CA certs for TLS connections to the database
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy only the migrate binary
COPY --from=builder /out/migrate /out/migrate

# Copy migrations to the well-known absolute path the binary checks first
COPY --from=builder /app/cmd/migrate/migrations /migrations

ENTRYPOINT ["/out/migrate"]

FROM scratch AS final

# Copy CA certificates so HTTPS calls inside the container work
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy compiled binaries
COPY --from=builder /out/api     /usr/local/bin/api
COPY --from=builder /out/migrate /usr/local/bin/migrate

# Copy migration files; the migrate binary resolves them at runtime via
COPY --from=builder /app/cmd/migrate/migrations /migrations

USER 65534:65534

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/api"]
