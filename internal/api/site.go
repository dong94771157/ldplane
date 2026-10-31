package api

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ldplane/ldplane/internal/config"
)

// Site 网站信息
type Site struct {
	Name      string `json:"name"`
	Domain    string `json:"domain"`
	Port      string `json:"port"`
	Root      string `json:"root"`
	Enabled   bool   `json:"enabled"`
	SSL       bool   `json:"ssl"`
	ConfPath  string `json:"conf_path"`
}

// SiteList 网站列表
func SiteList(c *gin.Context) {
	cfg := config.Get()
	sites := []Site{}

	entries, err := os.ReadDir(cfg.SiteDir)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": sites, "msg": "站点目录不存在"})
		return
	}

	enabledDir := "/etc/nginx/sites-enabled"
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
			continue
		}
		confPath := filepath.Join(cfg.SiteDir, e.Name())
		data, err := os.ReadFile(confPath)
		if err != nil {
			continue
		}
		site := parseSiteConf(e.Name(), confPath, string(data))
		// 检查是否启用
		enabledLink := filepath.Join(enabledDir, e.Name())
		if _, err := os.Stat(enabledLink); err == nil {
			site.Enabled = true
		}
		sites = append(sites, site)
	}

	sort.Slice(sites, func(i, j int) bool { return sites[i].Name < sites[j].Name })
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": sites})
}

// SiteCreate 创建网站
func SiteCreate(c *gin.Context) {
	var req struct {
		Name   string `json:"name"`
		Domain string `json:"domain"`
		Port   string `json:"port"`
		Root   string `json:"root"`
		SSL    bool   `json:"ssl"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	if req.Name == "" || req.Domain == "" || req.Port == "" || req.Root == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数不能为空"})
		return
	}

	cfg := config.Get()
	if err := os.MkdirAll(req.Root, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建网站根目录失败"})
		return
	}

	// 写入默认 index.html
	indexPath := filepath.Join(req.Root, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		content := fmt.Sprintf(`<html><body><h1>%s is running!</h1></body></html>`, req.Domain)
		_ = os.WriteFile(indexPath, []byte(content), 0644)
	}

	confName := req.Name + ".conf"
	confPath := filepath.Join(cfg.SiteDir, confName)
	confContent := buildSiteConf(req.Domain, req.Port, req.Root, req.SSL)
	if err := os.WriteFile(confPath, []byte(confContent), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "写入配置失败"})
		return
	}

	// 自动启用站点
	enabledLink := filepath.Join("/etc/nginx/sites-enabled", confName)
	_ = os.Symlink(confPath, enabledLink)

	if err := reloadNginx(); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "创建成功，但Nginx重载失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "创建成功"})
}

// SiteDelete 删除网站
func SiteDelete(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	cfg := config.Get()
	confName := req.Name + ".conf"
	confPath := filepath.Join(cfg.SiteDir, confName)
	enabledLink := filepath.Join("/etc/nginx/sites-enabled", confName)

	_ = os.Remove(enabledLink)
	if err := os.Remove(confPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除配置失败"})
		return
	}
	_ = reloadNginx()
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}

// SiteToggle 启用/禁用网站
func SiteToggle(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	cfg := config.Get()
	confName := req.Name + ".conf"
	confPath := filepath.Join(cfg.SiteDir, confName)
	enabledLink := filepath.Join("/etc/nginx/sites-enabled", confName)

	if req.Enabled {
		if err := os.Symlink(confPath, enabledLink); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "启用失败"})
			return
		}
	} else {
		_ = os.Remove(enabledLink)
	}
	_ = reloadNginx()
	msg := "禁用成功"
	if req.Enabled {
		msg = "启用成功"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": msg})
}

// SiteGetConf 获取网站配置
func SiteGetConf(c *gin.Context) {
	cfg := config.Get()
	name := c.Query("name")
	confPath := filepath.Join(cfg.SiteDir, name+".conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "读取配置失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": string(data)})
}

// SiteSaveConf 保存网站配置
func SiteSaveConf(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	cfg := config.Get()
	confPath := filepath.Join(cfg.SiteDir, req.Name+".conf")
	if err := os.WriteFile(confPath, []byte(req.Content), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存失败"})
		return
	}
	_ = reloadNginx()
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "保存成功"})
}

func reloadNginx() error {
	cfg := config.Get()
	cmd := exec.Command(cfg.NginxBin, "-t")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("nginx -t failed: %s", string(out))
	}
	cmd = exec.Command("systemctl", "reload", "nginx")
	return cmd.Run()
}

func parseSiteConf(name, confPath, content string) Site {
	site := Site{Name: strings.TrimSuffix(name, ".conf"), ConfPath: confPath}
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "server_name") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				site.Domain = strings.TrimSuffix(parts[1], ";")
			}
		}
		if strings.HasPrefix(line, "listen") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.Contains(p, ":") || isPort(p) {
					site.Port = strings.TrimSuffix(p, ";")
					break
				}
			}
			if strings.Contains(line, "ssl") {
				site.SSL = true
			}
		}
		if strings.HasPrefix(line, "root") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				site.Root = strings.TrimSuffix(parts[1], ";")
			}
		}
	}
	return site
}

func isPort(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

func buildSiteConf(domain, port, root string, ssl bool) string {
	conf := fmt.Sprintf(`server {
    listen %s;
    server_name %s;
    root %s;
    index index.html index.htm index.php;

    location / {
        try_files $uri $uri/ =404;
    }

    location ~ \.php$ {
        fastcgi_pass unix:/run/php/php-fpm.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }

    access_log /var/log/nginx/%s.access.log;
    error_log /var/log/nginx/%s.error.log;
}
`, port, domain, root, domain, domain)
	return conf
}