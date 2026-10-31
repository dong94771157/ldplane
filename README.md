# 琳东linux管理面板

琳东linux管理面板是一个类似宝塔面板的轻量级 Linux 服务器管理工具，使用 Go + Gin 开发，前端界面通过 `go:embed` 嵌入二进制，**单文件部署，开箱即用**。

## 功能特性

- **仪表盘** - CPU / 内存 / 磁盘 / 网络实时监控，系统信息一览
- **网站管理** - Nginx 站点创建 / 删除 / 启停 / 配置编辑
- **文件管理** - 浏览 / 上传 / 下载 / 编辑 / 重命名 / 删除 / 压缩 / 新建目录
- **服务管理** - systemd 服务列表 / 启动 / 停止 / 重启 / 重载
- **防火墙** - ufw / firewalld 规则添加 / 删除
- **进程管理** - 进程列表 / 结束进程 / 端口占用查看
- **Web 终端** - 浏览器内 WebSocket 终端（仅 Linux）
- **安全认证** - JWT Token + bcrypt 密码加密

## 快速开始

### 下载二进制

从 [Releases](../../releases) 页面下载对应架构的二进制文件：

```bash
# amd64
wget https://github.com/dong94771157/ldplane/releases/latest/download/ldplane-linux-amd64 -O ldplane

# arm64
wget https://github.com/dong94771157/ldplane/releases/latest/download/ldplane-linux-arm64 -O ldplane

chmod +x ldplane
./ldplane
```

### 从源码构建

```bash
git clone https://github.com/dong94771157/ldplane.git
cd ldplane
go build -o ldplane .
./ldplane
```

## 服务器部署（完整步骤）

以下命令在 Linux 服务器上以 root 执行：

### 1. 下载并安装

```bash
# 下载二进制（amd64 架构，arm64 请替换文件名）
wget https://github.com/dong94771157/ldplane/releases/latest/download/ldplane-linux-amd64 -O /usr/local/bin/ldplane
chmod +x /usr/local/bin/ldplane

# 创建配置目录
mkdir -p /etc/ldplane
```

### 2. 创建 systemd 服务

```bash
cat > /etc/systemd/system/ldplane.service << 'EOF'
[Unit]
Description=琳东linux管理面板
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/ldplane
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable ldplane
systemctl start ldplane
```

### 3. 查看状态与日志

```bash
systemctl status ldplane      # 查看运行状态
journalctl -u ldplane -f      # 实时查看日志
```

### 4. 开放防火墙端口

```bash
# ufw
ufw allow 8888/tcp

# 或 firewalld
firewall-cmd --permanent --add-port=8888/tcp
firewall-cmd --reload
```

### 5. 访问面板

浏览器打开 `http://服务器IP:8888`，默认账号 `admin` / `admin123`，**登录后立即修改密码**。

### 6. （可选）Nginx 反向代理 + HTTPS

```bash
apt install nginx certbot python3-certbot-nginx -y   # Debian/Ubuntu

cat > /etc/nginx/sites-available/ldplane << 'EOF'
server {
    listen 80;
    server_name panel.example.com;

    location / {
        proxy_pass http://127.0.0.1:8888;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 终端支持
    location /api/terminal/ws {
        proxy_pass http://127.0.0.1:8888;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400;
    }
}
EOF

ln -sf /etc/nginx/sites-available/ldplane /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx

# 申请 HTTPS 证书
certbot --nginx -d panel.example.com
```

### 7. 升级 / 卸载

```bash
# 升级
wget https://github.com/dong94771157/ldplane/releases/latest/download/ldplane-linux-amd64 -O /usr/local/bin/ldplane
chmod +x /usr/local/bin/ldplane
systemctl restart ldplane

# 卸载
systemctl stop ldplane && systemctl disable ldplane
rm -f /usr/local/bin/ldplane /etc/systemd/system/ldplane.service
rm -rf /etc/ldplane
systemctl daemon-reload
```

### 访问面板

启动后访问 `http://服务器IP:8888`

- 默认账号：`admin`
- 默认密码：`admin123`

> **重要**：首次登录后请立即在「设置」页面修改默认密码！

## 配置说明

配置文件位于 `/etc/ldplane/config.json`（非 root 用户运行时回退到 `./data/config.json`）：

```json
{
  "port": 8888,
  "username": "admin",
  "password": "$2a$10$...(bcrypt哈希)",
  "jwt_secret": "随机生成的密钥",
  "nginx_sites_dir": "/etc/nginx/sites-available"
}
```

首次运行会自动生成默认配置和随机 JWT 密钥。

## 系统要求

- Linux（推荐 Ubuntu / Debian / CentOS）
- Nginx（网站管理功能需要）
- systemd（服务管理功能需要）
- ufw 或 firewalld（防火墙功能需要，二选一）

> 非 Linux 系统可编译运行，但 Web 终端功能不可用。

## 技术栈

| 组件 | 说明 |
|------|------|
| Gin | HTTP 框架 |
| golang-jwt/v5 | JWT 认证 |
| gopsutil/v3 | 系统信息采集 |
| creack/pty | Web 终端伪终端 |
| gorilla/websocket | WebSocket 支持 |
| golang.org/x/crypto | bcrypt 密码加密 |

## 开源协议

MIT License