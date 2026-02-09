package shared

type AppScreen int

const (
	ScreenHome AppScreen = iota
	ScreenBrowse
)

func (s AppScreen) String() string {
	switch s {
	case ScreenHome:
		return "HOME"
	case ScreenBrowse:
		return "BROWSE"
	default:
		return "UNKNOWN"
	}
}

type Panel int

const (
	PanelSidebar Panel = iota
	PanelTable
	PanelEditor
)
