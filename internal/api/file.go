package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// FileItem 文件/目录信息
type FileItem struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"mod_time"`
}

// FileList 列出目录内容
func FileList(c *gin.Context) {
	dir := c.DefaultQuery("path", "/")
	if dir == "" {
		dir = "/"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无法读取目录: " + err.Error()})
		return
	}
	items := []FileItem{}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		items = append(items, FileItem{
			Name:    e.Name(),
			Path:    filepath.Join(dir, e.Name()),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			Mode:    info.Mode().String(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return items[i].Name < items[j].Name
	})
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"path":  dir,
		"items": items,
	}})
}

// FileDownload 下载文件
func FileDownload(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "路径不能为空"})
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "文件不存在"})
		return
	}
	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "不能下载目录"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(path))
	c.File(path)
}

// FileUpload 上传文件
func FileUpload(c *gin.Context) {
	dir := c.PostForm("path")
	if dir == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "目标目录不能为空"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "上传文件无效"})
		return
	}
	dst := filepath.Join(dir, file.Filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存文件失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "上传成功"})
}

// FileDelete 删除文件或目录
func FileDelete(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	// 安全检查：禁止删除根目录
	if req.Path == "/" || req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "禁止删除根目录"})
		return
	}
	if err := os.RemoveAll(req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}

// FileRename 重命名/移动
func FileRename(c *gin.Context) {
	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.OldPath == "" || req.NewPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	if err := os.Rename(req.OldPath, req.NewPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "重命名失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "重命名成功"})
}

// FileMkdir 创建目录
func FileMkdir(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	if err := os.MkdirAll(req.Path, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建目录失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "创建成功"})
}

// FileContent 获取文件内容（文本编辑）
func FileContent(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "路径不能为空"})
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "文件不存在"})
		return
	}
	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "不能读取目录内容"})
		return
	}
	// 限制文件大小 2MB
	if info.Size() > 2*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "文件过大，限制2MB以内"})
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "读取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": string(data)})
}

// FileSave 保存文件内容
func FileSave(c *gin.Context) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	if err := os.WriteFile(req.Path, []byte(req.Content), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "保存成功"})
}

// FileChmod 修改权限
func FileChmod(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" || req.Mode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	var mode uint32
	for _, ch := range req.Mode {
		if ch < '0' || ch > '7' {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "权限格式错误"})
			return
		}
		mode = mode*8 + uint32(ch-'0')
	}
	if err := os.Chmod(req.Path, os.FileMode(mode)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "修改权限失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "修改成功"})
}

// safePath 检查路径安全（防止目录穿越）
func safePath(base, target string) bool {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	return strings.HasPrefix(absTarget, absBase)
}

// FileCompress 压缩文件（tar.gz）- 简单实现
func FileCompress(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	// 使用系统 tar 命令
	dir := filepath.Dir(req.Path)
	name := filepath.Base(req.Path)
	outFile := req.Path + ".tar.gz"
	cmd := execCommand("tar", "-czf", outFile, "-C", dir, name)
	if out, err := cmd.CombinedOutput(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "压缩失败: " + string(out)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "压缩成功", "data": outFile})
}

// FileExtract 解压文件
func FileExtract(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	dir := filepath.Dir(req.Path)
	cmd := execCommand("tar", "-xzf", req.Path, "-C", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "解压失败: " + string(out)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "解压成功"})
}

// FileCopy 复制文件
func FileCopy(c *gin.Context) {
	var req struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Src == "" || req.Dst == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}
	srcFile, err := os.Open(req.Src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "打开源文件失败"})
		return
	}
	defer srcFile.Close()

	info, _ := srcFile.Stat()
	if info.IsDir() {
		// 目录复制使用 cp -r
		cmd := execCommand("cp", "-r", req.Src, req.Dst)
		if out, err := cmd.CombinedOutput(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "复制失败: " + string(out)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "复制成功"})
		return
	}

	dstFile, err := os.Create(req.Dst)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建目标文件失败"})
		return
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "复制失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "复制成功"})
}