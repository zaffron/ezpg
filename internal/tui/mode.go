package tui

import "github.com/zaffron/ezpg/internal/tui/shared"

// Re-export for convenience within the tui package
type AppScreen = shared.AppScreen
type Panel = shared.Panel

const (
	ScreenHome   = shared.ScreenHome
	ScreenBrowse = shared.ScreenBrowse
)

const (
	PanelSidebar = shared.PanelSidebar
	PanelTable   = shared.PanelTable
	PanelEditor  = shared.PanelEditor
)
