package keymaps

import "github.com/charmbracelet/bubbles/key"

type Keymap = struct {
	editMode, normalMode, ToggleFiles, openViewer, Quit, leader, selectFile key.Binding
}

func GetNormalKeyMaps() Keymap {
	return Keymap{
		normalMode: key.NewBinding(key.WithDisabled()),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "exit"),
		),
		leader: key.NewBinding(
			key.WithKeys("space"),
			key.WithHelp("space", "leader"),
		),
		ToggleFiles: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "files"),
		),
		openViewer: key.NewBinding(key.WithDisabled()),
		editMode: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "edit"),
		),
	}
}
