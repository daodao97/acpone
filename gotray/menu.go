package gotray

import (
	"github.com/getlantern/systray"
)

// MenuItem 表示一个菜单项
type MenuItem struct {
	Title    string
	Tooltip  string
	Icon     []byte
	OnClick  func(item *MenuItem)
	Disabled bool
	Hidden   bool

	sysItem *systray.MenuItem
}

// SetTitle 设置菜单标题
func (m *MenuItem) SetTitle(title string) {
	m.Title = title
	if m.sysItem != nil {
		m.sysItem.SetTitle(title)
	}
}

// SetTooltip 设置菜单提示
func (m *MenuItem) SetTooltip(tooltip string) {
	m.Tooltip = tooltip
	if m.sysItem != nil {
		m.sysItem.SetTooltip(tooltip)
	}
}

// Enable 启用菜单项
func (m *MenuItem) Enable() {
	m.Disabled = false
	if m.sysItem != nil {
		m.sysItem.Enable()
	}
}

// Disable 禁用菜单项
func (m *MenuItem) Disable() {
	m.Disabled = true
	if m.sysItem != nil {
		m.sysItem.Disable()
	}
}

// Show 显示菜单项
func (m *MenuItem) Show() {
	m.Hidden = false
	if m.sysItem != nil {
		m.sysItem.Show()
	}
}

// Hide 隐藏菜单项
func (m *MenuItem) Hide() {
	m.Hidden = true
	if m.sysItem != nil {
		m.sysItem.Hide()
	}
}

// Check 勾选菜单项
func (m *MenuItem) Check() {
	if m.sysItem != nil {
		m.sysItem.Check()
	}
}

// Uncheck 取消勾选
func (m *MenuItem) Uncheck() {
	if m.sysItem != nil {
		m.sysItem.Uncheck()
	}
}

// Checked 是否被勾选
func (m *MenuItem) Checked() bool {
	if m.sysItem != nil {
		return m.sysItem.Checked()
	}
	return false
}

// AddMenu 添加普通菜单项
func (a *App) AddMenu(title string, onClick func(item *MenuItem)) *MenuItem {
	return a.AddMenuWithOptions(&MenuItem{
		Title:   title,
		OnClick: onClick,
	})
}

// AddMenuWithOptions 添加带选项的菜单项
func (a *App) AddMenuWithOptions(item *MenuItem) *MenuItem {
	sysItem := systray.AddMenuItem(item.Title, item.Tooltip)
	item.sysItem = sysItem

	if len(item.Icon) > 0 {
		sysItem.SetIcon(item.Icon)
	}
	if item.Disabled {
		sysItem.Disable()
	}
	if item.Hidden {
		sysItem.Hide()
	}

	go func() {
		for range sysItem.ClickedCh {
			if item.OnClick != nil {
				item.OnClick(item)
			}
		}
	}()

	a.menus = append(a.menus, item)
	return item
}

// AddCheckbox 添加复选框菜单项
func (a *App) AddCheckbox(title string, checked bool, onClick func(item *MenuItem)) *MenuItem {
	sysItem := systray.AddMenuItemCheckbox(title, "", checked)
	item := &MenuItem{
		Title:   title,
		OnClick: onClick,
		sysItem: sysItem,
	}

	go func() {
		for range sysItem.ClickedCh {
			if item.OnClick != nil {
				item.OnClick(item)
			}
		}
	}()

	a.menus = append(a.menus, item)
	return item
}

// AddSeparator 添加分隔线
func (a *App) AddSeparator() {
	systray.AddSeparator()
}

// MenuGroup 菜单组
type MenuGroup struct {
	Title   string
	Items   []*MenuItem
	sysItem *systray.MenuItem
}

// AddGroup 添加菜单组
func (a *App) AddGroup(title string, items []*MenuItem) *MenuGroup {
	parent := systray.AddMenuItem(title, "")
	group := &MenuGroup{
		Title:   title,
		Items:   items,
		sysItem: parent,
	}

	for _, item := range items {
		subItem := parent.AddSubMenuItem(item.Title, item.Tooltip)
		item.sysItem = subItem

		itemRef := item
		go func() {
			for range subItem.ClickedCh {
				if itemRef.OnClick != nil {
					itemRef.OnClick(itemRef)
				}
			}
		}()
	}

	return group
}

// RadioGroup 单选菜单组
type RadioGroup struct {
	Title    string
	Items    []*MenuItem
	Selected int
	sysItem  *systray.MenuItem
}

// AddRadioGroup 添加单选菜单组
func (a *App) AddRadioGroup(title string, defaultIdx int, items []*MenuItem) *RadioGroup {
	parent := systray.AddMenuItem(title, "")
	group := &RadioGroup{
		Title:    title,
		Items:    items,
		Selected: defaultIdx,
		sysItem:  parent,
	}

	var sysItems []*systray.MenuItem

	for i, item := range items {
		checked := i == defaultIdx
		subItem := parent.AddSubMenuItemCheckbox(item.Title, item.Tooltip, checked)
		item.sysItem = subItem
		sysItems = append(sysItems, subItem)

		idx := i
		itemRef := item
		go func() {
			for range subItem.ClickedCh {
				// 更新选中状态
				for j, si := range sysItems {
					if j == idx {
						si.Check()
					} else {
						si.Uncheck()
					}
				}
				group.Selected = idx

				if itemRef.OnClick != nil {
					itemRef.OnClick(itemRef)
				}
			}
		}()
	}

	return group
}

// AddQuitMenu 添加退出菜单项
func (a *App) AddQuitMenu(title string, onQuit func()) *MenuItem {
	return a.AddMenu(title, func(item *MenuItem) {
		if onQuit != nil {
			onQuit()
		}
		a.Quit()
	})
}
