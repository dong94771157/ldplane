package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 更新源地址（R2 镜像）
const UpdateBaseURL = "https://mb.ldnet.tech"

// VersionInfo 远端版本信息
type VersionInfo struct {
	Version   string            `json:"version"`
	Date      string            `json:"date"`
	Changelog []string          `json:"changelog"`
	Downloads map[string]string `json:"downloads"`
}

// UpdateCheck 检查更新
func UpdateCheck(c *gin.Context) {
	info, err := fetchRemoteVersion()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"current_version": CurrentVersion,
				"has_update":      false,
				"error":           "无法获取版本信息: " + err.Error(),
			},
		})
		return
	}
	hasUpdate := compareVersion(info.Version, CurrentVersion) > 0
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"current_version": CurrentVersion,
			"latest_version":  info.Version,
			"has_update":      hasUpdate,
			"date":            info.Date,
			"changelog":       info.Changelog,
			"download_url":    getDownloadURL(info),
		},
	})
}

// UpdateApply 执行更新
func UpdateApply(c *gin.Context) {
	info, err := fetchRemoteVersion()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取版本信息失败: " + err.Error()})
		return
	}
	if compareVersion(info.Version, CurrentVersion) <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "当前已是最新版本"})
		return
	}

	downloadURL := getDownloadURL(info)
	if downloadURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "未找到当前架构的下载地址"})
		return
	}

	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取程序路径失败: " + err.Error()})
		return
	}
	exePath, _ = filepath.EvalSymlinks(exePath)

	// 下载新版本到临时文件
	tmpPath := exePath + ".new"
	if err := downloadFile(downloadURL, tmpPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "下载失败: " + err.Error()})
		return
	}

	// 校验文件大小（基本防错）
	if fi, err := os.Stat(tmpPath); err != nil || fi.Size() < 1024*1024 {
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "下载文件异常，已取消更新"})
		return
	}

	// 赋予执行权限
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "设置权限失败: " + err.Error()})
		return
	}

	// 备份旧版本
	bakPath := exePath + ".bak"
	os.Remove(bakPath)
	if err := os.Rename(exePath, bakPath); err != nil {
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "备份旧版本失败: " + err.Error()})
		return
	}

	// 替换为新版本
	if err := os.Rename(tmpPath, exePath); err != nil {
		// 回滚
		os.Rename(bakPath, exePath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "替换文件失败，已回滚: " + err.Error()})
		return
	}

	// 异步重启
	go func() {
		time.Sleep(1 * time.Second)
		// 优先 systemctl restart
		if err := exec.Command("systemctl", "restart", "ldplane").Run(); err != nil {
			// 回退: 直接 exec 自身
			_ = exec.Command(exePath).Start()
			os.Exit(0)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "更新完成，服务正在重启..."})
}

// fetchRemoteVersion 从 R2 获取版本信息
func fetchRemoteVersion() (*VersionInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(UpdateBaseURL + "/version.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	info := &VersionInfo{}
	if err := json.Unmarshal(data, info); err != nil {
		return nil, err
	}
	return info, nil
}

// getDownloadURL 根据当前架构获取下载地址
func getDownloadURL(info *VersionInfo) string {
	if info.Downloads != nil {
		if u, ok := info.Downloads[runtime.GOARCH]; ok {
			return u
		}
	}
	// 回退: 拼接默认路径
	return fmt.Sprintf("%s/ldplane-linux-%s", UpdateBaseURL, runtime.GOARCH)
}

// downloadFile 下载文件
func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// compareVersion 比较版本号，返回 >0 表示 a 更新
func compareVersion(a, b string) int {
	// 去掉 v 前缀
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	maxLen := len(as)
	if len(bs) > maxLen {
		maxLen = len(bs)
	}
	for i := 0; i < maxLen; i++ {
		var av, bv int
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av != bv {
			if av > bv {
				return 1
			}
			return -1
		}
	}
	return 0
}