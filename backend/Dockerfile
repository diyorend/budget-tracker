# Multi-stage build — the pattern the job listing asked for
FROM golang:1.22-alpine AS builder

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
