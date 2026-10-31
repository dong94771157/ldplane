package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ldplane/ldplane/internal/middleware"
)

//go:embed all:web_dist
var webFS embed.FS

// SetupRouter 设置路由
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// API 路由组
	api := r.Group("/api")
	{
		// 无需认证
		api.POST("/login", Login)

		// 需要认证
		auth := api.Group("", middleware.AuthRequired())
		{
			auth.GET("/user", GetUserInfo)
			auth.POST("/password", ChangePassword)

			// 系统监控
			auth.GET("/system/overview", SystemOverview)
			auth.GET("/system/info", SystemInfo)

			// 网站管理
			auth.GET("/site/list", SiteList)
			auth.POST("/site/create", SiteCreate)
			auth.POST("/site/delete", SiteDelete)
			auth.POST("/site/toggle", SiteToggle)
			auth.GET("/site/conf", SiteGetConf)
			auth.POST("/site/conf", SiteSaveConf)

			// 文件管理
			auth.GET("/file/list", FileList)
			auth.GET("/file/download", FileDownload)
			auth.POST("/file/upload", FileUpload)
			auth.POST("/file/delete", FileDelete)
			auth.POST("/file/rename", FileRename)
			auth.POST("/file/mkdir", FileMkdir)
			auth.GET("/file/content", FileContent)
			auth.POST("/file/save", FileSave)
			auth.POST("/file/chmod", FileChmod)
			auth.POST("/file/compress", FileCompress)
			auth.POST("/file/extract", FileExtract)
			auth.POST("/file/copy", FileCopy)

			// 服务管理
			auth.GET("/service/list", ServiceList)
			auth.POST("/service/action", ServiceAction)

			// 防火墙
			auth.GET("/firewall/list", FirewallList)
			auth.POST("/firewall/add", FirewallAdd)
			auth.POST("/firewall/delete", FirewallDelete)
			auth.GET("/firewall/status", FirewallStatus)

			// 进程管理
			auth.GET("/process/list", ProcessList)
			auth.POST("/process/kill", ProcessKill)
			auth.GET("/process/port", PortList)

			// 终端
			auth.GET("/terminal/ws", TerminalWS)
		}
	}

	// 静态文件服务
	staticFS, err := fs.Sub(webFS, "web_dist")
	if err == nil {
		r.GET("/", func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", mustReadFile(staticFS, "index.html"))
		})
		r.GET("/favicon.ico", func(c *gin.Context) {
			c.Data(http.StatusOK, "image/x-icon", mustReadFile(staticFS, "favicon.ico"))
		})
		r.NoRoute(func(c *gin.Context) {
			// SPA 回退到 index.html
			c.Data(http.StatusOK, "text/html; charset=utf-8", mustReadFile(staticFS, "index.html"))
		})
		r.StaticFS("/static", http.FS(staticFS))
	}

	return r
}

func mustReadFile(fsys fs.FS, name string) []byte {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return []byte("Not Found")
	}
	return data
}