package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CurrentVersion 当前版本号
// 由 main 包通过 SetVersion 设置（构建时 -ldflags 注入），默认 1.0.1
var CurrentVersion = "1.0.1"

// SetVersion 设置版本号
func SetVersion(v string) {
	if v != "" && v != "dev" {
		CurrentVersion = v
	}
}

// GetVersion 获取当前版本
func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"version": CurrentVersion}})
}