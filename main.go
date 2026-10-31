package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/ldplane/ldplane/internal/api"
	"github.com/ldplane/ldplane/internal/config"
)

var version = "dev"

func setGinMode(mode string) {
	gin.SetMode(mode)
}

func main() {
	// 加载配置
	cfg := config.Load()

	// 设置 gin 模式
	ginMode := "release"
	if cfg.Port == 0 {
		ginMode = "debug"
	}
	setGinMode(ginMode)

	// 设置路由
	r := api.SetupRouter()

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("琳东linux管理面板 v%s 启动中...", version)
	log.Printf("监听地址: http://0.0.0.0%s", addr)
	log.Printf("默认账号: %s (请及时修改密码)", cfg.Username)

	if err := r.Run(addr); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}