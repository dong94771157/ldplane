package api

import (
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// FirewallRule 防火墙规则
type FirewallRule struct {
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
	Action   string `json:"action"`
	Comment  string `json:"comment"`
}

// FirewallList 防火墙规则列表
func FirewallList(c *gin.Context) {
	rules := []FirewallRule{}

	// 优先尝试 ufw
	if out, err := exec.Command("ufw", "status", "numbered").Output(); err == nil {
		rules = parseUfwStatus(string(out))
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": rules, "type": "ufw"})
		return
	}

	// 尝试 firewalld
	if out, err := exec.Command("firewall-cmd", "--list-ports").Output(); err == nil {
		rules = parseFirewalldPorts(string(out))
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": rules, "type": "firewalld"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rules, "type": "none", "msg": "未检测到防火墙(ufw/firewalld)"})
}

// FirewallAdd 添加防火墙规则
func FirewallAdd(c *gin.Context) {
	var req struct {
		Port     string `json:"port"`
		Protocol string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Port == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	if req.Protocol == "" {
		req.Protocol = "tcp"
	}
	if _, err := strconv.Atoi(req.Port); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "端口必须是数字"})
		return
	}

	// 尝试 ufw
	if out, err := exec.Command("ufw", "allow", req.Port+"/"+req.Protocol).CombinedOutput(); err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "添加成功"})
		return
	} else {
		_ = out
	}

	// 尝试 firewalld
	cmd := exec.Command("firewall-cmd", "--permanent", "--add-port="+req.Port+"/"+req.Protocol)
	if _, err := cmd.Output(); err == nil {
		_ = exec.Command("firewall-cmd", "--reload").Run()
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "添加成功"})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "添加失败，请检查防火墙是否安装"})
}

// FirewallDelete 删除防火墙规则
func FirewallDelete(c *gin.Context) {
	var req struct {
		Port     string `json:"port"`
		Protocol string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Port == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	if req.Protocol == "" {
		req.Protocol = "tcp"
	}

	// 尝试 ufw
	if out, err := exec.Command("ufw", "delete", "allow", req.Port+"/"+req.Protocol).CombinedOutput(); err == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
		return
	} else {
		_ = out
	}

	// 尝试 firewalld
	cmd := exec.Command("firewall-cmd", "--permanent", "--remove-port="+req.Port+"/"+req.Protocol)
	if _, err := cmd.Output(); err == nil {
		_ = exec.Command("firewall-cmd", "--reload").Run()
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除失败"})
}

// FirewallStatus 防火墙状态
func FirewallStatus(c *gin.Context) {
	// 尝试 ufw
	if out, err := exec.Command("ufw", "status").Output(); err == nil {
		active := strings.Contains(string(out), "Status: active")
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"type": "ufw", "active": active}})
		return
	}
	// 尝试 firewalld
	if out, err := exec.Command("firewall-cmd", "--state").Output(); err == nil {
		active := strings.TrimSpace(string(out)) == "running"
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"type": "firewalld", "active": active}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"type": "none", "active": false}})
}

func parseUfwStatus(output string) []FirewallRule {
	rules := []FirewallRule{}
	re := regexp.MustCompile(`\[\s*(\d+)\]\s+(\S+)\s+(ALLOW|DENY)\s+(.+?)(?:#\s*(.*))?$`)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 5 {
			portProto := matches[2]
			port, proto := portProto, "tcp"
			if idx := strings.Index(portProto, "/"); idx > 0 {
				port = portProto[:idx]
				proto = portProto[idx+1:]
			}
			comment := ""
			if len(matches) >= 6 {
				comment = strings.TrimSpace(matches[5])
			}
			rules = append(rules, FirewallRule{
				Port:     port,
				Protocol: proto,
				Action:   matches[3],
				Comment:  comment,
			})
		}
	}
	return rules
}

func parseFirewalldPorts(output string) []FirewallRule {
	rules := []FirewallRule{}
	for _, p := range strings.Fields(strings.TrimSpace(output)) {
		port, proto := p, "tcp"
		if idx := strings.Index(p, "/"); idx > 0 {
			port = p[:idx]
			proto = p[idx+1:]
		}
		rules = append(rules, FirewallRule{Port: port, Protocol: proto, Action: "ALLOW"})
	}
	return rules
}