package homescreen

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaffron/ezpg/internal/tui/shared"
)

func (h HomeScreen) formView() string {
	var b strings.Builder

	heading := "Create Connection"
	if h.editing {
		heading = "Edit Connection"
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(shared.ColorPrimary).
		Render(heading)
	b.WriteString("\n" + title + "\n\n")

	separator := lipgloss.NewStyle().
		Foreground(shared.ColorMuted).
		Render("  -- or use URL to connect directly --")

	for i, f := range h.fields {
		if i == int(fieldURL) {
			b.WriteString("\n" + separator + "\n\n")
		}
		cursor := "  "
		if i == int(h.activeField) {
			cursor = "> "
		}
		b.WriteString(cursor + f.View() + "\n")
	}

	content := b.String()
	style := lipgloss.NewStyle().
		Width(h.width).
		Height(h.height).
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render(content)
}

func (h HomeScreen) listView() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(shared.ColorPrimary).
		Render("lazygres")
	b.WriteString("\n" + title + "\n\n")

	connTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(shared.ColorFg).
		Render("Connections:")
	b.WriteString(connTitle + "\n\n")

	if len(h.connections) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(shared.ColorMuted).
			Render("  No connections configured. Press 'c' to create one.")
		b.WriteString(empty + "\n")
	} else {
		for i, conn := range h.connections {
			prefix := "  "
			if i == h.cursor {
				prefix = "> "
			}

			name := conn.Name
			var detail string
			if conn.URL != "" {
				detail = conn.URL
				// Truncate long URLs
				if len(detail) > 50 {
					detail = detail[:47] + "..."
				}
			} else {
				host := conn.Host
				if conn.Port != 0 && conn.Port != 5432 {
					host = fmt.Sprintf("%s:%d", host, conn.Port)
				} else if conn.Port == 0 {
					host = fmt.Sprintf("%s:5432", host)
				} else {
					host = fmt.Sprintf("%s:%d", host, conn.Port)
				}
				if conn.Database != "" {
					host += "/" + conn.Database
				}
				detail = host
			}

			nameStyle := lipgloss.NewStyle().Foreground(shared.ColorFg)
			detailStyle := lipgloss.NewStyle().Foreground(shared.ColorMuted)
			if i == h.cursor {
				nameStyle = nameStyle.Foreground(shared.ColorPrimary).Bold(true)
			}

			// Pad name to align details
			paddedName := fmt.Sprintf("%-20s", name)
			line := prefix + nameStyle.Render(paddedName) + detailStyle.Render(detail)
			b.WriteString(line + "\n")
		}
	}

	// Center the content
	content := b.String()
	style := lipgloss.NewStyle().
		Width(h.width).
		Height(h.height).
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render(content)
}

func (h HomeScreen) View() string {
	if h.editing || h.creating {
		return h.formView()
	}
	return h.listView()
}
