package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ldplane/ldplane/internal/config"
	"github.com/ldplane/ldplane/internal/middleware"
)

// Login 用户登录
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	cfg := config.Get()
	if req.Username != cfg.Username || !config.CheckPassword(cfg.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "用户名或密码错误"})
		return
	}
	token, err := middleware.GenerateToken(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "生成令牌失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "登录成功", "data": gin.H{"token": token}})
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	cfg := config.Get()
	if !config.CheckPassword(cfg.PasswordHash, req.OldPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "旧密码错误"})
		return
	}
	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "新密码长度不能少于6位"})
		return
	}
	cfg.PasswordHash = config.HashPassword(req.NewPassword)
	if err := cfg.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "密码修改成功"})
}

// GetUserInfo 获取当前用户信息
func GetUserInfo(c *gin.Context) {
	cfg := config.Get()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"username": cfg.Username,
	}})
}