FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata chromium \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

ENV CHROME_BIN=/usr/bin/chromium-browser
ENV TZ=Asia/Shanghai

COPY --from=builder /app/server /app/server

USER appuser

EXPOSE 28131

ENTRYPOINT ["/app/server"]
