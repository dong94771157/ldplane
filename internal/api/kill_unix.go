//go:build !windows

package api

import "syscall"

// killProcess 结束进程（Unix 系统：先 SIGTERM 再 SIGKILL）
func killProcess(pid int) error {
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		// 尝试强制杀死
		if err2 := syscall.Kill(pid, syscall.SIGKILL); err2 != nil {
			return err
		}
	}
	return nil
}