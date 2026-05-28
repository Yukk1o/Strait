# 构建阶段
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o strait ./cmd/strait/

# 运行阶段
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/strait .
COPY --from=builder /app/configs ./configs

EXPOSE 8080
CMD ["./strait"]