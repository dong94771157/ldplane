# ============ 构建阶段 ============
FROM golang:1.23-alpine AS builder

WORKDIR /build

# 先复制依赖文件，利用 Docker 缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建二进制（版本号通过 build-arg 传入）
ARG VERSION=1.0.1
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o ldplane .

# ============ 运行阶段 ============
FROM alpine:3.20

# 安装运行时依赖（nginx 用于网站管理，systemd 不可用则跳过）
RUN apk add --no-cache ca-certificates tzdata nginx \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

# 复制二进制
COPY --from=builder /build/ldplane .

# 创建数据目录
RUN mkdir -p /etc/ldplane /etc/nginx/conf.d /etc/nginx/sites-available

# 暴露端口
EXPOSE 8888

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8888/ || exit 1

ENTRYPOINT ["./ldplane"]