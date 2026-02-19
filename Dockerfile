# === Build Stage ===
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/newsbot ./cmd/newsbot && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/devkit ./cmd/devkit && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bin/watchbot ./cmd/watchbot

# === Runtime Stage ===
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=builder /bin/newsbot /bin/newsbot
COPY --from=builder /bin/devkit /bin/devkit
COPY --from=builder /bin/watchbot /bin/watchbot

ENTRYPOINT ["/bin/newsbot"]
CMD ["serve"]
