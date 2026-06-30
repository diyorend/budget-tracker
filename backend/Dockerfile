# Multi-stage build.
#
# IMPORTANT: the builder image's Go version must be >= the "go" directive in
# go.mod (currently 1.26.4). If they mismatch, `go mod download` tries to
# silently fetch the right toolchain from proxy.golang.org at build time —
# and in most sandboxed/offline Docker build environments that download is
# blocked, which is exactly what produced the original build failure:
#   "go mod download" did not complete successfully: exit code: 1
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 = static binary, no .so deps
# -ldflags "-s -w" = strip debug info, smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server/main.go

# Final stage — minimal image
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

# Non-root user for security
RUN addgroup -g 1001 app && adduser -u 1001 -G app -D app
USER app

COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
