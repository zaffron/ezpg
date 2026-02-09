package homescreen

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/zaffron/ezpg/internal/config"
)

// Form
type formField int

const (
	fieldName formField = iota
	fieldHost
	fieldPort
	fieldUser
	fieldPassword
	fieldDatabase
	fieldSSLMode
	fieldURL
	fieldCount
)

// HomeScreen
type HomeScreen struct {
	connections []config.Connection
	cursor      int
	width       int
	height      int

	// Form state
	editing     bool
	creating    bool
	editIdx     int // index into connections when editing
	fields      []textinput.Model
	activeField formField
}
