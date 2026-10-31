package api

import (
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/process"
)

// ProcessItem 进程信息
type ProcessItem struct {
	Pid     int32   `json:"pid"`
	Name    string  `json:"name"`
	Cpu     float64 `json:"cpu"`
	Memory  float32 `json:"memory"`
	Status  string  `json:"status"`
	User    string  `json:"user"`
	Command string  `json:"command"`
}

// ProcessList 进程列表
func ProcessList(c *gin.Context) {
	procs, err := process.Processes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取进程失败"})
		return
	}
	items := []ProcessItem{}
	for _, p := range procs {
		name, _ := p.Name()
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		status, _ := p.Status()
		user, _ := p.Username()
		cmdline, _ := p.Cmdline()
		if cmdline == "" {
			cmdline = name
		}
		// 截断过长的命令行
		if len(cmdline) > 200 {
			cmdline = cmdline[:200] + "..."
		}
		items = append(items, ProcessItem{
			Pid:     p.Pid,
			Name:    name,
			Cpu:     cpu,
			Memory:  mem,
			Status:  strings.Join(status, " "),
			User:    user,
			Command: cmdline,
		})
	}
	// 按 CPU 排序
	sort.Slice(items, func(i, j int) bool { return items[i].Cpu > items[j].Cpu })
	// 限制返回前100个
	if len(items) > 100 {
		items = items[:100]
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": items})
}

// ProcessKill 结束进程
func ProcessKill(c *gin.Context) {
	var req struct {
		Pid int `json:"pid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Pid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	// 禁止杀死 init 进程
	if req.Pid == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "禁止结束系统进程"})
		return
	}
	if err := killProcess(req.Pid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "结束进程失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "进程已结束"})
}

// PortList 端口占用列表
func PortList(c *gin.Context) {
	// 使用 ss 命令查看端口
	cmd := exec.Command("ss", "-tlnp")
	out, err := cmd.Output()
	if err != nil {
		// 回退到 netstat
		cmd = exec.Command("netstat", "-tlnp")
		out, err = cmd.Output()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": []interface{}{}, "msg": "无法获取端口信息"})
			return
		}
	}
	ports := parsePortOutput(string(out))
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": ports})
}

type PortInfo struct {
	Proto     string `json:"proto"`
	LocalAddr string `json:"local_addr"`
	Port      string `json:"port"`
	State     string `json:"state"`
	Process   string `json:"process"`
}

func parsePortOutput(output string) []PortInfo {
	ports := []PortInfo{}
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if i < 1 { // 跳过标题行
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		proto := fields[0]
		localAddr := fields[3]
		state := fields[1]
		proc := ""
		if len(fields) >= 6 {
			proc = strings.Join(fields[5:], " ")
		}
		// 提取端口
		port := ""
		if idx := strings.LastIndex(localAddr, ":"); idx > 0 {
			port = localAddr[idx+1:]
		}
		if port == "" {
			continue
		}
		ports = append(ports, PortInfo{
			Proto:     proto,
			LocalAddr: localAddr,
			Port:      port,
			State:     state,
			Process:   proc,
		})
	}
	return ports
}

// execCmd 执行命令（用于其他地方）
func execCmd(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// strToInt 字符串转整数
func strToInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}