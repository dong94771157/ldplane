//go:build windows

package api

import (
	"fmt"
	"os"
	"os/exec"
)

// killProcess 结束进程（Windows 系统：使用 taskkill 强制结束进程树）
func killProcess(pid int) error {
	// 优先使用 taskkill 结束进程树
	if out, err := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", pid)).CombinedOutput(); err == nil {
		return nil
	} else {
		_ = out
		// 回退到 os.FindProcess
		p, err2 := os.FindProcess(pid)
		if err2 != nil {
			return err
		}
		return p.Kill()
	}
}