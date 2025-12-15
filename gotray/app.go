package gotray

import (
	"runtime"

	"github.com/getlantern/systray"
)

// App 是系统托盘应用的核心结构
type App struct {
	Name        string
	DisplayName string
	Identifier  string
	Version     string

	// 图标 (PNG 格式用于 macOS/Linux, ICO 格式用于 Windows)
	Icon    []byte
	IconOff []byte
	IconWin []byte
	IconOffWin []byte

	// 生命周期回调
	OnReady func(app *App)
	OnExit  func()

	// 内部状态
	menus []*MenuItem
}

// Run 启动应用
func (a *App) Run() {
	systray.Run(func() {
		a.setIcon(a.Icon, a.IconWin)
		if a.OnReady != nil {
			a.OnReady(a)
		}
	}, func() {
		if a.OnExit != nil {
			a.OnExit()
		}
	})
}

// SetIcon 设置托盘图标
func (a *App) SetIcon(icon []byte) {
	a.setIcon(icon, a.IconWin)
}

// SetIconOn 设置激活状态图标
func (a *App) SetIconOn() {
	a.setIcon(a.Icon, a.IconWin)
}

// SetIconOff 设置未激活状态图标
func (a *App) SetIconOff() {
	a.setIcon(a.IconOff, a.IconOffWin)
}

func (a *App) setIcon(icon, iconWin []byte) {
	if IsWindows() && len(iconWin) > 0 {
		systray.SetTemplateIcon(iconWin, iconWin)
	} else if len(icon) > 0 {
		systray.SetTemplateIcon(icon, icon)
	}
}

// SetTooltip 设置托盘提示文字
func (a *App) SetTooltip(tooltip string) {
	systray.SetTooltip(tooltip)
}

// SetTitle 设置托盘标题 (仅 macOS)
func (a *App) SetTitle(title string) {
	systray.SetTitle(title)
}

// Quit 退出应用
func (a *App) Quit() {
	systray.Quit()
}

// OS 返回当前操作系统名称 (darwin, linux, windows)
func OS() string {
	return runtime.GOOS
}

// IsWindows 检查是否为 Windows 系统
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS 检查是否为 macOS 系统
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux 检查是否为 Linux 系统
func IsLinux() bool {
	return runtime.GOOS == "linux"
}
