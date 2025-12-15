package main

import (
	_ "embed"
	"fmt"

	"github.com/daodao97/acpone/gotray"
)

//go:embed icon.png
var icon []byte

func main() {
	app := &gotray.App{
		Name:        "Example",
		DisplayName: "GoTray Example",
		Identifier:  "com.example.gotray",
		Version:     "1.0.0",
		Icon:        icon,
		OnReady:     onReady,
		OnExit: func() {
			fmt.Println("App exit")
		},
	}

	app.Run()
}

func onReady(app *gotray.App) {
	app.SetTooltip("GoTray Example App")

	// 普通菜单
	app.AddMenu("Open Dashboard", func(item *gotray.MenuItem) {
		gotray.OpenURL("http://localhost:3000")
	})

	// 状态切换菜单
	statusMenu := app.AddMenu("Start Service", func(item *gotray.MenuItem) {
		if item.Title == "Start Service" {
			// 启动服务逻辑
			item.SetTitle("Stop Service")
			app.SetIconOn()
			gotray.NotifySimple("Service", "Service started")
		} else {
			// 停止服务逻辑
			item.SetTitle("Start Service")
			app.SetIconOff()
			gotray.NotifySimple("Service", "Service stopped")
		}
	})
	_ = statusMenu

	app.AddSeparator()

	// 复选框菜单
	autoStart := gotray.NewAutoStart("gotray-example", "GoTray Example")
	app.AddCheckbox("Launch at Login", autoStart.IsEnabled(), func(item *gotray.MenuItem) {
		if item.Checked() {
			item.Uncheck()
			autoStart.Disable()
		} else {
			item.Check()
			autoStart.Enable()
		}
	})

	// 菜单组
	app.AddGroup("About", []*gotray.MenuItem{
		{Title: "GitHub", OnClick: func(item *gotray.MenuItem) {
			gotray.OpenURL("https://github.com")
		}},
		{Title: "Documentation", OnClick: func(item *gotray.MenuItem) {
			gotray.OpenURL("https://example.com/docs")
		}},
	})

	app.AddSeparator()

	// 退出菜单
	app.AddQuitMenu("Quit", func() {
		fmt.Println("Cleaning up...")
	})
}
