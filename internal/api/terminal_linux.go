//go:build linux

package api

import (
	"net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var terminalUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// TerminalWS Web 终端 WebSocket 处理
func TerminalWS(c *gin.Context) {
	conn, err := terminalUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 创建 shell 进程
	cmd := exec.Command(os.Getenv("SHELL"))
	if cmd.Path == "" {
		cmd = exec.Command("/bin/bash")
	}
	cmd.Env = append(os.Environ(), "TERM=xterm")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("无法创建终端: "+err.Error()))
		return
	}
	defer func() {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
	}()

	// 读取终端输出 -> WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// 读取 WebSocket -> 终端输入
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
			if _, err := ptmx.Write(msg); err != nil {
				return
			}
		}
	}
}