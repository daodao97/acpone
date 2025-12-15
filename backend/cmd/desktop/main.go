package main

import (
	_ "embed"
	"fmt"
	"net"

	"github.com/anthropics/acpone/gotray"
	"github.com/anthropics/acpone/internal/api"
	"github.com/anthropics/acpone/internal/config"
	"github.com/anthropics/acpone/web"
)

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
