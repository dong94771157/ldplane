#!/bin/bash
# 琳东linux管理面板 一键安装脚本
#
# 用法:
#   1. 默认(GitHub下载):  curl -fsSL https://raw.githubusercontent.com/dong94771157/ldplane/master/install.sh | bash
#   2. 自定义下载域名:    curl -fsSL https://raw.githubusercontent.com/dong94771157/ldplane/master/install.sh | bash -s -- https://www.ldnet.tech/dl
#   3. 环境变量方式:      DOWNLOAD_BASE=https://www.ldnet.tech/dl bash install.sh
#
# 自定义域名目录下需放置以下文件:
#   ldplane-linux-amd64  (x86_64架构二进制)
#   ldplane-linux-arm64  (arm64架构二进制)

set -e

REPO="dong94771157/ldplane"
DEFAULT_BASE="https://github.com/${REPO}/releases/latest/download"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/ldplane"
SERVICE_FILE="/etc/systemd/system/ldplane.service"
PORT=8888

# 下载域名: 优先级 命令行参数 > 环境变量 > 默认GitHub
if [ -n "$1" ]; then
    DOWNLOAD_BASE="$1"
elif [ -n "$DOWNLOAD_BASE" ]; then
    DOWNLOAD_BASE="$DOWNLOAD_BASE"
else
    DOWNLOAD_BASE="$DEFAULT_BASE"
fi

# 检查 root
if [ "$(id -u)" != "0" ]; then
    echo "错误: 请以 root 用户运行此脚本"
    exit 1
fi

# 检测架构
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  BINARY="ldplane-linux-amd64" ;;
    aarch64|arm64) BINARY="ldplane-linux-arm64" ;;
    *)
        echo "错误: 不支持的架构 $ARCH"
        exit 1
        ;;
esac

echo "========================================"
echo "  琳东linux管理面板 - 一键安装"
echo "========================================"
echo "架构: $ARCH"
echo "二进制: $BINARY"
echo "下载源: $DOWNLOAD_BASE"
echo ""

# 下载最新版本
echo "[1/5] 下载 琳东linux管理面板 ..."
DOWNLOAD_URL="${DOWNLOAD_BASE}/${BINARY}"
if command -v wget &> /dev/null; then
    wget -q --show-progress -O "${INSTALL_DIR}/ldplane" "$DOWNLOAD_URL"
elif command -v curl &> /dev/null; then
    curl -fSL -o "${INSTALL_DIR}/ldplane" "$DOWNLOAD_URL"
else
    echo "错误: 需要 wget 或 curl"
    exit 1
fi
chmod +x "${INSTALL_DIR}/ldplane"
echo "下载完成: ${INSTALL_DIR}/ldplane"

# 创建配置目录
echo "[2/5] 创建配置目录 ..."
mkdir -p "$CONFIG_DIR"

# 创建 systemd 服务
echo "[3/5] 配置 systemd 服务 ..."
cat > "$SERVICE_FILE" << EOF
[Unit]
Description=琳东linux管理面板
After=network.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/ldplane
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
echo "[4/5] 启动服务 ..."
systemctl daemon-reload
systemctl enable ldplane
systemctl start ldplane

# 开放防火墙
echo "[5/5] 配置防火墙 ..."
if command -v ufw &> /dev/null && ufw status | grep -q "active"; then
    ufw allow ${PORT}/tcp
    echo "ufw 已开放 ${PORT} 端口"
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-port=${PORT}/tcp
    firewall-cmd --reload
    echo "firewalld 已开放 ${PORT} 端口"
else
    echo "未检测到活跃防火墙，跳过端口开放"
fi

echo ""
echo "========================================"
echo "  安装完成!"
echo "========================================"
echo ""
echo "  访问地址: http://$(hostname -I | awk '{print $1}'):${PORT}"
echo "  默认账号: admin"
echo "  默认密码: admin123"
echo ""
echo "  重要: 登录后请立即在「设置」中修改默认密码!"
echo ""
echo "  常用命令:"
echo "    systemctl status ldplane    # 查看状态"
echo "    systemctl restart ldplane   # 重启服务"
echo "    journalctl -u ldplane -f    # 查看日志"
echo ""