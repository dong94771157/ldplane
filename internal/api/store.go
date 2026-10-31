package api

import (
	"net/http"
	"os/exec"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// SoftItem 软件商店条目
type SoftItem struct {
	Name        string `json:"name"`         // 显示名称
	Pkg         string `json:"pkg"`          // 包名
	Category    string `json:"category"`     // 分类: database/web/runtime/tool
	Version     string `json:"version"`      // 推荐版本
	Description string `json:"description"`  // 描述
	Installed   bool   `json:"installed"`    // 是否已安装
	Active      bool   `json:"active"`       // 服务是否运行中
	Installing  bool   `json:"installing"`   // 是否安装中
}

// 软件目录
var softCatalog = []SoftItem{
	// ===== 数据库 =====
	{Name: "MySQL 8.0", Pkg: "mysql-server", Category: "database", Version: "8.0", Description: "最流行的开源关系型数据库"},
	{Name: "MariaDB", Pkg: "mariadb-server", Category: "database", Version: "10.6", Description: "MySQL 的社区维护分支，完全兼容"},
	{Name: "Redis", Pkg: "redis-server", Category: "database", Version: "7.0", Description: "高性能内存键值数据库/缓存"},
	{Name: "PostgreSQL", Pkg: "postgresql", Category: "database", Version: "15", Description: "功能强大的开源对象关系型数据库"},
	{Name: "MongoDB", Pkg: "mongodb-org", Category: "database", Version: "7.0", Description: "面向文档的 NoSQL 数据库"},
	{Name: "SQLite", Pkg: "sqlite3", Category: "database", Version: "3", Description: "轻量级嵌入式 SQL 数据库引擎"},
	{Name: "Memcached", Pkg: "memcached", Category: "database", Version: "1.6", Description: "高性能分布式内存缓存系统"},
	// ===== Web 服务 =====
	{Name: "Nginx", Pkg: "nginx", Category: "web", Version: "1.24", Description: "高性能 HTTP 和反向代理服务器"},
	{Name: "Apache", Pkg: "apache2", Category: "web", Version: "2.4", Description: "世界上最流行的 Web 服务器"},
	{Name: "OpenResty", Pkg: "openresty", Category: "web", Version: "1.21", Description: "基于 Nginx 的动态 Web 平台"},
	// ===== 运行时 =====
	{Name: "PHP 8.2", Pkg: "php8.2-fpm", Category: "runtime", Version: "8.2", Description: "PHP 8.2 运行时 + FPM"},
	{Name: "Node.js 20", Pkg: "nodejs", Category: "runtime", Version: "20", Description: "JavaScript 服务端运行时"},
	{Name: "Python 3", Pkg: "python3", Category: "runtime", Version: "3.11", Description: "Python 3 运行时环境"},
	{Name: "Go", Pkg: "golang-go", Category: "runtime", Version: "1.22", Description: "Go 语言工具链"},
	{Name: "Java (OpenJDK)", Pkg: "openjdk-17-jdk", Category: "runtime", Version: "17", Description: "OpenJDK 17 Java 运行环境"},
	// ===== 工具 =====
	{Name: "Docker", Pkg: "docker.io", Category: "tool", Version: "24", Description: "容器化应用运行平台"},
	{Name: "Git", Pkg: "git", Category: "tool", Version: "2.39", Description: "分布式版本控制系统"},
	{Name: "Supervisor", Pkg: "supervisor", Category: "tool", Version: "4.2", Description: "进程管理工具"},
	{Name: "Fail2ban", Pkg: "fail2ban", Category: "tool", Version: "0.11", Description: "入侵防御框架"},
	{Name: "Certbot", Pkg: "certbot", Category: "tool", Version: "2.1", Description: "Let's Encrypt SSL 证书工具"},
}

// 安装任务状态
var (
	installTasks = make(map[string]bool) // pkg -> installing
	taskMu       sync.Mutex
)

// SoftList 软件商店列表
func SoftList(c *gin.Context) {
	category := c.Query("category")
	items := []SoftItem{}
	for _, s := range softCatalog {
		if category != "" && s.Category != category {
			continue
		}
		item := s
		item.Installed = isPkgInstalled(s.Pkg)
		item.Active = isServiceActive(s.Pkg)
		taskMu.Lock()
		item.Installing = installTasks[s.Pkg]
		taskMu.Unlock()
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// SoftInstall 安装软件
func SoftInstall(c *gin.Context) {
	var req struct {
		Pkg string `json:"pkg"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Pkg == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	// 校验包名在目录中
	valid := false
	for _, s := range softCatalog {
		if s.Pkg == req.Pkg {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "未知软件包"})
		return
	}
	if isPkgInstalled(req.Pkg) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已安装"})
		return
	}
	taskMu.Lock()
	if installTasks[req.Pkg] {
		taskMu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": "正在安装中，请稍候"})
		return
	}
	installTasks[req.Pkg] = true
	taskMu.Unlock()

	// 异步安装
	go func(pkg string) {
		defer func() {
			taskMu.Lock()
			delete(installTasks, pkg)
			taskMu.Unlock()
		}()
		_ = exec.Command("bash", "-c",
			"export DEBIAN_FRONTEND=noninteractive; apt-get update -qq && apt-get install -y -qq "+pkg).Run()
	}(req.Pkg)

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "安装任务已提交，后台执行中"})
}

// SoftUninstall 卸载软件
func SoftUninstall(c *gin.Context) {
	var req struct {
		Pkg string `json:"pkg"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Pkg == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	if !isPkgInstalled(req.Pkg) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "未安装"})
		return
	}
	cmd := exec.Command("bash", "-c",
		"export DEBIAN_FRONTEND=noninteractive; apt-get purge -y -qq "+req.Pkg+" && apt-get autoremove -y -qq")
	if out, err := cmd.CombinedOutput(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "卸载失败: " + strings.TrimSpace(string(out))})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "卸载成功"})
}

// isPkgInstalled 检查包是否已安装
func isPkgInstalled(pkg string) bool {
	out, err := exec.Command("dpkg", "-s", pkg).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Status: install ok installed")
}

// isServiceActive 检查对应服务是否运行
func isServiceActive(pkg string) bool {
	svcName := pkgToService(pkg)
	if svcName == "" {
		return false
	}
	out, err := exec.Command("systemctl", "is-active", svcName).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

// pkgToService 包名转服务名
func pkgToService(pkg string) string {
	m := map[string]string{
		"mysql-server":  "mysql",
		"mariadb-server": "mariadb",
		"redis-server":  "redis",
		"postgresql":    "postgresql",
		"mongodb-org":   "mongod",
		"memcached":     "memcached",
		"nginx":         "nginx",
		"apache2":       "apache2",
		"php8.2-fpm":    "php8.2-fpm",
		"docker.io":     "docker",
		"supervisor":    "supervisor",
		"fail2ban":      "fail2ban",
	}
	if svc, ok := m[pkg]; ok {
		return svc
	}
	return pkg
}