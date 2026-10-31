//go:build !linux

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TerminalWS Web 终端（非 Linux 平台不可用）
func TerminalWS(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"code": 501, "msg": "终端功能仅在 Linux 上可用"})
}