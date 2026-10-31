package api

import "os/exec"

// execCommand 执行系统命令
func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}