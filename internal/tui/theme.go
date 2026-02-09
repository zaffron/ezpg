package tui

import "github.com/zaffron/ezpg/internal/tui/shared"

// Re-export styles and colors for convenience XD

var (
	ColorPrimary   = shared.ColorPrimary
	ColorSecondary = shared.ColorSecondary
	ColorSuccess   = shared.ColorSuccess
	ColorDanger    = shared.ColorDanger
	ColorWarning   = shared.ColorWarning
	ColorMuted     = shared.ColorMuted
	ColorBg        = shared.ColorBg
	ColorBgAlt     = shared.ColorBgAlt
	ColorFg        = shared.ColorFg
	ColorBorder    = shared.ColorBorder

	StyleSidebarActive   = shared.StyleSidebarActive
	StyleSidebarInactive = shared.StyleSidebarInactive
	StyleMainActive      = shared.StyleMainActive
	StyleMainInactive    = shared.StyleMainInactive
	StyleEditorActive    = shared.StyleEditorActive
	StyleEditorInactive  = shared.StyleEditorInactive
	StyleKeyHintKey      = shared.StyleKeyHintKey
	StyleKeyHintDesc     = shared.StyleKeyHintDesc
)
