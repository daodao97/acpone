package main

import (
	_ "embed"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/acpone/gotray"
	"github.com/anthropics/acpone/internal/api"
	"github.com/anthropics/acpone/internal/config"
	"github.com/anthropics/acpone/web"
)

func init() {
	// GUI 应用不会继承终端的 PATH，需要手动设置
	setupPath()
}

func setupPath() {
	home, _ := os.UserHomeDir()
	if home == "" {
		return
	}

	var extraPaths []string
	var pathSep string

	switch gotray.OS() {
	case "windows":
		pathSep = ";"
		extraPaths = getWindowsPaths(home)
	case "darwin":
		pathSep = ":"
		extraPaths = getMacOSPaths(home)
	default: // linux
		pathSep = ":"
		extraPaths = getLinuxPaths(home)
	}

	// 获取当前 PATH
	currentPath := os.Getenv("PATH")
	pathSet := make(map[string]bool)
	for _, p := range strings.Split(currentPath, pathSep) {
		pathSet[p] = true
	}

	// 添加新路径（如果存在且不重复）
	var newPaths []string
	for _, p := range extraPaths {
		if _, err := os.Stat(p); err == nil && !pathSet[p] {
			newPaths = append(newPaths, p)
			pathSet[p] = true
		}
	}

	if len(newPaths) > 0 {
		finalPath := strings.Join(newPaths, pathSep) + pathSep + currentPath
		os.Setenv("PATH", finalPath)
	}
}

func getMacOSPaths(home string) []string {
	paths := []string{
		"/usr/local/bin",
		"/opt/homebrew/bin",                       // Homebrew (Apple Silicon)
		"/opt/homebrew/sbin",
		filepath.Join(home, ".local", "bin"),      // pipx, etc.
		filepath.Join(home, ".cargo", "bin"),      // Rust
		filepath.Join(home, "go", "bin"),          // Go
		filepath.Join(home, ".npm-global", "bin"), // npm global
		filepath.Join(home, ".bun", "bin"),        // Bun
	}

	// nvm 安装的 node
	paths = append(paths, findNodeVersions(filepath.Join(home, ".nvm", "versions", "node"), "bin")...)

	// fnm 安装的 node
	fnmDir := filepath.Join(home, "Library", "Application Support", "fnm", "node-versions")
	paths = append(paths, findNodeVersions(fnmDir, "installation", "bin")...)

	return paths
}

func getLinuxPaths(home string) []string {
	paths := []string{
		"/usr/local/bin",
		"/snap/bin",                               // Snap packages
		filepath.Join(home, ".local", "bin"),      // pipx, etc.
		filepath.Join(home, ".cargo", "bin"),      // Rust
		filepath.Join(home, "go", "bin"),          // Go
		filepath.Join(home, ".npm-global", "bin"), // npm global
		filepath.Join(home, ".bun", "bin"),        // Bun
	}

	// nvm 安装的 node
	paths = append(paths, findNodeVersions(filepath.Join(home, ".nvm", "versions", "node"), "bin")...)

	// fnm 安装的 node (Linux)
	fnmDir := filepath.Join(home, ".local", "share", "fnm", "node-versions")
	paths = append(paths, findNodeVersions(fnmDir, "installation", "bin")...)

	return paths
}

func getWindowsPaths(home string) []string {
	appData := os.Getenv("APPDATA")
	localAppData := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("ProgramFiles")

	paths := []string{
		filepath.Join(programFiles, "nodejs"),           // Node.js
		filepath.Join(appData, "npm"),                   // npm global
		filepath.Join(localAppData, "Programs", "Python", "Python311", "Scripts"), // Python
		filepath.Join(localAppData, "Programs", "Python", "Python312", "Scripts"),
		filepath.Join(home, ".cargo", "bin"),            // Rust
		filepath.Join(home, "go", "bin"),                // Go
		filepath.Join(home, ".bun", "bin"),              // Bun
		filepath.Join(appData, "fnm", "node-versions"),  // fnm
	}

	// nvm-windows
	nvmHome := os.Getenv("NVM_HOME")
	if nvmHome != "" {
		paths = append(paths, nvmHome)
		nvmSymlink := os.Getenv("NVM_SYMLINK")
		if nvmSymlink != "" {
			paths = append(paths, nvmSymlink)
		}
	}

	// fnm 安装的 node (Windows)
	fnmDir := filepath.Join(appData, "fnm", "node-versions")
	paths = append(paths, findNodeVersions(fnmDir, "installation")...)

	return paths
}

func findNodeVersions(baseDir string, subPaths ...string) []string {
	var paths []string
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return paths
	}
	for _, entry := range entries {
		if entry.IsDir() {
			parts := append([]string{baseDir, entry.Name()}, subPaths...)
			paths = append(paths, filepath.Join(parts...))
		}
	}
	return paths
}

//go:embed icon/icon.png
var icon []byte

//go:embed icon/icon_off.png
var iconOff []byte

const (
	appName       = "ACPone"
	appIdentifier = "com.anthropic.acpone"
	appVersion    = "1.0.0"
	defaultPort   = "3000"
)

var (
	server    *api.Server
	isRunning bool
	serverURL string
)

func main() {
	app := &gotray.App{
		Name:        appName,
		DisplayName: appName,
		Identifier:  appIdentifier,
		Version:     appVersion,
		Icon:        icon,
		IconOff:     iconOff,
		OnReady:     onReady,
		OnExit:      onExit,
	}

	app.Run()
}

func onReady(app *gotray.App) {
	app.SetTooltip(appName + " - ACP Gateway")

	// 打开浏览器菜单
	openMenu := app.AddMenu("Open Dashboard", func(item *gotray.MenuItem) {
		if serverURL != "" {
			gotray.OpenURL(serverURL)
		}
	})

	app.AddSeparator()

	// 启动/停止服务菜单
	serviceMenu := app.AddMenu("Start Server", func(item *gotray.MenuItem) {
		item.Disable()
		defer item.Enable()

		if isRunning {
			stopServer()
			item.SetTitle("Start Server")
			app.SetIconOff()
			openMenu.Disable()
			gotray.NotifySimple(appName, "Server stopped")
		} else {
			if err := startServer(); err != nil {
				gotray.NotifySimple(appName, "Failed to start: "+err.Error())
				return
			}
			item.SetTitle("Stop Server")
			app.SetIconOn()
			openMenu.Enable()
			gotray.NotifySimple(appName, "Server started at "+serverURL)
		}
	})

	// 自动启动服务器
	if err := startServer(); err != nil {
		gotray.NotifySimple(appName, "Failed to start: "+err.Error())
		serviceMenu.SetTitle("Start Server")
		openMenu.Disable()
	} else {
		serviceMenu.SetTitle("Stop Server")
		app.SetIconOn()
		gotray.NotifySimple(appName, "Server started at "+serverURL)
	}

	app.AddSeparator()

	// 开机启动
	autoStart := gotray.NewAutoStart("acpone", appName)
	app.AddCheckbox("Launch at Login", autoStart.IsEnabled(), func(item *gotray.MenuItem) {
		if item.Checked() {
			item.Uncheck()
			_ = autoStart.Disable()
		} else {
			item.Check()
			_ = autoStart.Enable()
		}
	})

	// 打开配置文件
	app.AddMenu("Edit Config", func(item *gotray.MenuItem) {
		configPath := config.LoadedConfigPath
		if configPath == "" {
			configPath = config.FindConfigPath()
		}
		if configPath != "" {
			_ = gotray.OpenWithApp(configPath, "Visual Studio Code")
		}
	})

	app.AddSeparator()

	// 关于菜单
	app.AddGroup("About", []*gotray.MenuItem{
		{Title: "GitHub", OnClick: func(item *gotray.MenuItem) {
			gotray.OpenURL("https://github.com/anthropics/acpone")
		}},
		{Title: "Documentation", OnClick: func(item *gotray.MenuItem) {
			gotray.OpenURL("https://github.com/anthropics/acpone#readme")
		}},
	})

	app.AddSeparator()

	// 退出菜单
	app.AddQuitMenu("Quit", func() {
		stopServer()
	})
}

func onExit() {
	stopServer()
	fmt.Println("ACPone exited")
}

func startServer() error {
	// 确保配置存在
	if err := config.EnsureConfigExists(); err != nil {
		fmt.Printf("Config initialization warning: %v\n", err)
	}

	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	// 获取静态文件
	staticFS, _ := web.FS()

	// 查找可用端口
	port := findAvailablePort(defaultPort)
	serverURL = fmt.Sprintf("http://localhost:%s", port)

	// 创建并启动服务器
	server = api.NewServer(cfg, staticFS)

	go func() {
		addr := ":" + port
		if err := server.ListenAndServe(addr); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	isRunning = true
	return nil
}

func stopServer() {
	if server != nil {
		server.Shutdown()
		server = nil
	}
	isRunning = false
	serverURL = ""
}

func findAvailablePort(preferred string) string {
	// 尝试首选端口
	if isPortAvailable(preferred) {
		return preferred
	}

	// 查找其他可用端口
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return preferred
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("%d", addr.Port)
}

func isPortAvailable(port string) bool {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}
