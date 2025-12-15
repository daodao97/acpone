package gotray

import (
	"os/exec"
	"runtime"
)

// Notification 通知消息
type Notification struct {
	Title   string
	Message string
	Sender  string // macOS bundle identifier
}

// Notify 发送系统通知
func Notify(n *Notification) error {
	switch runtime.GOOS {
	case "darwin":
		return notifyMacOS(n)
	case "windows":
		return notifyWindows(n)
	case "linux":
		return notifyLinux(n)
	default:
		return nil
	}
}

// NotifySimple 发送简单通知
func NotifySimple(title, message string) error {
	return Notify(&Notification{
		Title:   title,
		Message: message,
	})
}

func notifyMacOS(n *Notification) error {
	script := `display notification "` + n.Message + `" with title "` + n.Title + `"`
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func notifyWindows(n *Notification) error {
	// Windows 通知需要 PowerShell
	script := `
	[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
	[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
	$template = @"
	<toast>
		<visual>
			<binding template="ToastText02">
				<text id="1">` + n.Title + `</text>
				<text id="2">` + n.Message + `</text>
			</binding>
		</visual>
	</toast>
"@
	$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
	$xml.LoadXml($template)
	$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
	[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("GoTray").Show($toast)
	`
	cmd := exec.Command("powershell", "-Command", script)
	return cmd.Run()
}

func notifyLinux(n *Notification) error {
	cmd := exec.Command("notify-send", n.Title, n.Message)
	return cmd.Run()
}
