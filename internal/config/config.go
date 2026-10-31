package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
)

// Config 面板配置
type Config struct {
	Port         int    `json:"port"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	JwtSecret    string `json:"jwt_secret"`
	DataDir      string `json:"data_dir"`
	NginxConfDir string `json:"nginx_conf_dir"`
	NginxBin     string `json:"nginx_bin"`
	SiteDir      string `json:"site_dir"`
}

var cfg *Config

// Load 加载配置，不存在则创建默认配置
func Load() *Config {
	dataDir := getDataDir()
	confPath := filepath.Join(dataDir, "config.json")

	data, err := os.ReadFile(confPath)
	if err == nil {
		cfg = &Config{}
		if json.Unmarshal(data, cfg) == nil {
			return cfg
		}
	}

	// 创建默认配置
	cfg = &Config{
		Port:         8888,
		Username:     "admin",
		PasswordHash: hashPassword("admin123"),
		JwtSecret:    randomHex(32),
		DataDir:      dataDir,
		NginxConfDir: "/etc/nginx/conf.d",
		NginxBin:     "nginx",
		SiteDir:      "/etc/nginx/sites-available",
	}
	_ = cfg.Save()
	return cfg
}

// Get 获取当前配置
func Get() *Config {
	if cfg == nil {
		cfg = Load()
	}
	return cfg
}

// Save 保存配置到文件
func (c *Config) Save() error {
	if err := os.MkdirAll(c.DataDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.DataDir, "config.json"), data, 0600)
}

func getDataDir() string {
	// 优先使用 /etc/ldplane，失败则用当前目录 ./data
	if _, err := os.Stat("/etc/ldplane"); err == nil {
		return "/etc/ldplane"
	}
	if os.Geteuid() == 0 {
		_ = os.MkdirAll("/etc/ldplane", 0755)
		return "/etc/ldplane"
	}
	return "./data"
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// HashPassword 生成密码哈希
func HashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

// CheckPassword 校验密码
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func hashPassword(password string) string {
	return HashPassword(password)
}