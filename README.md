# LDPlane - 轻量级 Linux 服务器管理面板

LDPlane 是一个类似宝塔面板的轻量级 Linux 服务器管理工具，使用 Go + Gin 开发，前端界面通过 `go:embed` 嵌入二进制，**单文件部署，开箱即用**。

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
wget https://github.com/ldplane/ldplane/releases/latest/download/ldplane-linux-amd64 -O ldplane

# arm64
wget https://github.com/ldplane/ldplane/releases/latest/download/ldplane-linux-arm64 -O ldplane

chmod +x ldplane
./ldplane
```

### 从源码构建

```bash
git clone https://github.com/ldplane/ldplane.git
cd ldplane
go build -o ldplane .
./ldplane
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