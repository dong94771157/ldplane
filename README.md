# 琳东linux管理面板

琳东linux管理面板是一个类似宝塔面板的轻量级 Linux 服务器管理工具，使用 Go + Gin 开发，前端界面通过 `go:embed` 嵌入二进制，**单文件部署，开箱即用**。

## 功能特性

- **仪表盘** - CPU / 内存 / 磁盘 / 网络实时监控，系统信息一览
- **网站管理** - Nginx 站点创建 / 删除 / 启停 / 配置编辑
- **文件管理** - 浏览 / 上传 / 下载 / 编辑 / 重命名 / 删除 / 压缩 / 新建目录
- **软件商店** - MySQL / Redis / PostgreSQL / MongoDB 等一键安装 / 卸载
- **服务管理** - systemd 服务列表 / 启动 / 停止 / 重启 / 重载
- **防火墙** - ufw / firewalld 规则添加 / 删除
- **进程管理** - 进程列表 / 结束进程 / 端口占用查看
- **Web 终端** - 浏览器内 WebSocket 终端（仅 Linux）
- **在线更新** - 面板内一键检查并更新到最新版本
- **Docker 部署** - 提供 Dockerfile，支持容器化运行
- **安全认证** - JWT Token + bcrypt 密码加密

## 部署方式

### 方式一：一键安装脚本（推荐）

```bash
# 默认从 GitHub Releases 下载
bash <(curl -sL https://raw.githubusercontent.com/dong94771157/ldplane/master/install.sh)

# 国内用户推荐：使用官方镜像加速
bash <(curl -sL https://raw.githubusercontent.com/dong94771157/ldplane/master/install.sh) https://mb.ldnet.tech

# 或通过环境变量指定下载源
export DOWNLOAD_BASE=https://mb.ldnet.tech
bash <(curl -sL https://raw.githubusercontent.com/dong94771157/ldplane/master/install.sh)
```

> 下载源优先级：命令行参数 > 环境变量 `DOWNLOAD_BASE` > 默认 GitHub Releases

脚本自动完成：下载二进制 → 安装到 `/usr/local/bin` → 创建 systemd 服务 → 启动服务 → 开放防火墙端口。

### 方式二：Docker 部署

```bash
# 构建镜像
docker build -t ldplane:1.0.1 .

# 运行容器
docker run -d \
  --name ldplane \
  -p 8888:8888 \
  -v ldplane-data:/etc/ldplane \
  -v nginx-conf:/etc/nginx/conf.d \
  --restart unless-stopped \
  ldplane:1.0.1
```

> Docker 环境下服务管理（systemd）和防火墙功能受限，仪表盘、文件管理、软件商店、Web 终端等正常可用。

### 方式三：手动部署

```bash
# 1. 下载二进制（国内镜像推荐）
wget https://mb.ldnet.tech/ldplane-linux-amd64 -O /usr/local/bin/ldplane
# arm64 架构请使用 ldplane-linux-arm64
chmod +x /usr/local/bin/ldplane

# 2. 创建 systemd 服务
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

# 3. 开放防火墙端口
ufw allow 8888/tcp          # ufw
# 或
firewall-cmd --permanent --add-port=8888/tcp && firewall-cmd --reload   # firewalld
```

### 方式四：从源码构建

```bash
git clone https://github.com/dong94771157/ldplane.git
cd ldplane
go build -trimpath -ldflags="-s -w -X main.version=1.0.1" -o ldplane .
./ldplane
```

## 访问面板

启动后访问 `http://服务器IP:8888`

- 默认账号：`admin`
- 默认密码：`admin123`

> **重要**：首次登录后请立即在「设置」页面修改默认密码！

## 升级与卸载

### 在线更新（推荐）

登录面板 → **设置** → **系统更新** → 检查更新 → 立即更新

面板会自动从官方源下载最新版本并重启服务，全程无需 SSH 操作。

### 手动升级

```bash
wget https://mb.ldnet.tech/ldplane-linux-amd64 -O /usr/local/bin/ldplane
chmod +x /usr/local/bin/ldplane
systemctl restart ldplane
```

### 卸载

```bash
systemctl stop ldplane && systemctl disable ldplane
rm -f /usr/local/bin/ldplane /etc/systemd/system/ldplane.service
rm -rf /etc/ldplane
systemctl daemon-reload
```

## （可选）Nginx 反向代理 + HTTPS

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

## 配置说明

配置文件位于 `/etc/ldplane/config.json`（非 root 用户运行时回退到 `./data/config.json`）：

```json
{
  "port": 8888,
  "username": "admin",
  "password_hash": "$2a$10$...(bcrypt哈希)",
  "jwt_secret": "随机生成的密钥",
  "data_dir": "/etc/ldplane",
  "nginx_conf_dir": "/etc/nginx/conf.d",
  "nginx_bin": "nginx",
  "site_dir": "/etc/nginx/sites-available"
}
```

首次运行会自动生成默认配置和随机 JWT 密钥。

## 系统要求

- Linux（推荐 Ubuntu / Debian / CentOS）
- Nginx（网站管理功能需要）
- systemd（服务管理功能需要）
- ufw 或 firewalld（防火墙功能需要，二选一）

> 非 Linux 系统可编译运行，但 Web 终端功能不可用。

## 更新日志

### v1.0.1 (2026-10-31)

- 新增**软件商店**：MySQL / MariaDB / Redis / PostgreSQL / MongoDB / SQLite / Memcached 等一键安装 / 卸载
- 新增**在线更新**：设置页检查并一键更新到最新版本（更新源：mb.ldnet.tech）
- 新增 **Docker 部署**：提供多阶段构建 Dockerfile，支持容器化运行
- 新增官方镜像下载源 `mb.ldnet.tech`（Cloudflare R2），国内访问加速
- 优化界面与交互体验

### v1.0.0

- 首个正式版本：仪表盘 / 网站管理 / 文件管理 / 服务管理 / 防火墙 / 进程管理 / Web 终端

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