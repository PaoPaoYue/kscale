# 构建阶段
FROM golang:1.23.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .

COPY --from=builder /app/.env .

CMD ["./main"]
