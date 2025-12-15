package gotray

import (
	"os"
	"path/filepath"

	"github.com/emersion/go-autostart"
)

// AutoStart 开机启动管理
type AutoStart struct {
	app     *autostart.App
	appName string
}

// NewAutoStart 创建开机启动管理器
func NewAutoStart(name, displayName string) *AutoStart {
	execPath, _ := os.Executable()

	var execCmd []string
	if IsMacOS() {
		// macOS 使用 open -a 打开应用
		execCmd = []string{"open", "-a", displayName}
	} else {
		execCmd = []string{execPath}
	}

	return &AutoStart{
		appName: name,
		app: &autostart.App{
			Name:        name,
			DisplayName: displayName,
			Exec:        execCmd,
		},
	}
}

// NewAutoStartWithPath 创建带自定义路径的开机启动管理器
func NewAutoStartWithPath(name, displayName, execPath string) *AutoStart {
	return &AutoStart{
		appName: name,
		app: &autostart.App{
			Name:        name,
			DisplayName: displayName,
			Exec:        []string{execPath},
		},
	}
}

// IsEnabled 检查是否已启用开机启动
func (a *AutoStart) IsEnabled() bool {
	return a.app.IsEnabled()
}

// Enable 启用开机启动
func (a *AutoStart) Enable() error {
	if a.IsEnabled() {
		return nil
	}
	return a.app.Enable()
}

// Disable 禁用开机启动
func (a *AutoStart) Disable() error {
	if !a.IsEnabled() {
		return nil
	}
	return a.app.Disable()
}

// Toggle 切换开机启动状态
func (a *AutoStart) Toggle() (bool, error) {
	if a.IsEnabled() {
		err := a.Disable()
		return false, err
	}
	err := a.Enable()
	return true, err
}

// GetConfigDir 获取应用配置目录
func GetConfigDir(appName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var configDir string
	switch {
	case IsMacOS():
		configDir = filepath.Join(home, "Library", "Application Support", appName)
	case IsWindows():
		configDir = filepath.Join(os.Getenv("APPDATA"), appName)
	default:
		configDir = filepath.Join(home, ".config", appName)
	}

	// 确保目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}

// GetDataDir 获取应用数据目录
func GetDataDir(appName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var dataDir string
	switch {
	case IsMacOS():
		dataDir = filepath.Join(home, "Library", "Application Support", appName)
	case IsWindows():
		dataDir = filepath.Join(os.Getenv("LOCALAPPDATA"), appName)
	default:
		dataDir = filepath.Join(home, ".local", "share", appName)
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", err
	}

	return dataDir, nil
}
