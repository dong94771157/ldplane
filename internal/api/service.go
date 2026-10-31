package api

import (
	"net/http"
	"os/exec"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// ServiceItem 服务信息
type ServiceItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LoadState   string `json:"load_state"`
	ActiveState string `json:"active_state"`
	SubState    string `json:"sub_state"`
}

// ServiceList 服务列表
func ServiceList(c *gin.Context) {
	// 获取常见服务列表
	commonServices := []string{
		"nginx", "apache2", "httpd", "mysql", "mariadb", "postgresql",
		"redis", "mongod", "php-fpm", "ssh", "sshd", "firewalld",
		"ufw", "docker", "cron", "rsyslog", "systemd-resolved",
		"networking", "vsftpd", "smbd", "supervisor", "fail2ban",
	}
	keyword := c.Query("keyword")
	if keyword != "" {
		commonServices = []string{keyword}
	}

	services := []ServiceItem{}
	for _, name := range commonServices {
		svc := getServiceStatus(name)
		if svc != nil {
			services = append(services, *svc)
		}
	}
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": services})
}

// ServiceAction 服务操作 (start/stop/restart/reload)
func ServiceAction(c *gin.Context) {
	var req struct {
		Name   string `json:"name"`
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" || req.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	validActions := map[string]bool{"start": true, "stop": true, "restart": true, "reload": true}
	if !validActions[req.Action] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效操作"})
		return
	}
	cmd := exec.Command("systemctl", req.Action, req.Name)
	if out, err := cmd.CombinedOutput(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": req.Action + "失败: " + strings.TrimSpace(string(out))})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": req.Action + "成功"})
}

func getServiceStatus(name string) *ServiceItem {
	cmd := exec.Command("systemctl", "show", name,
		"--property=Id,Description,LoadState,ActiveState,SubState,UnitFileState")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	svc := &ServiceItem{}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "Id":
			svc.Name = parts[1]
		case "Description":
			svc.Description = parts[1]
		case "LoadState":
			svc.LoadState = parts[1]
		case "ActiveState":
			svc.ActiveState = parts[1]
		case "SubState":
			svc.SubState = parts[1]
		}
	}
	// 未安装的服务跳过
	if svc.LoadState == "not-found" || svc.Name == "" {
		return nil
	}
	return svc
}