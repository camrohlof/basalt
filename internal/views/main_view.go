package mainview

import (
	"camrohlof/basalt/internal/components/editor"
	"camrohlof/basalt/internal/keymaps"
	"camrohlof/basalt/internal/utils"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mistakenelf/teacup/statusbar"
)

type state int

const (
	edit state = iota
	files
	tooSmall
	initalizing
)

func (s state) String() string {
	switch s {
	case edit:
		return "edit"
	case files:
		return "files"
	case tooSmall:
		return "too small"
	case initalizing:
		return "initalizing"
	default:
		return "huh?"
	}
}

type Model struct {
	config    utils.Config
	textarea  editor.Model
	filelist  list.Model
	statusbar statusbar.Model
	height    int
	width     int
	keymap    keymaps.Keymap
	help      help.Model
	contents  string
	state     state
}

var (
	modelStyle        = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).BorderStyle(lipgloss.HiddenBorder())
	focusedModelStyle = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).BorderStyle(lipgloss.NormalBorder())
)

func getFirstFile(lastFile string) string {
	out, _ := os.ReadFile(lastFile)
	return string(out)
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func getFileTree(root string) []list.Item {
	var items []list.Item
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Fatal(err)
	}

	for _, ele := range entries {
		var item item
		item.title = ele.Name()
		if ele.IsDir() {
			item.desc = "Directory"
			items = append(items, item)
		} else {
			if string(ele.Name()[len(ele.Name())-3:]) == ".md" {
				item.desc = "File"
				items = append(items, item)
			}
		}

	}
	return items
}

func New(cfg utils.Config) Model {
	file := getFirstFile(cfg.LastFile)
	ta := editor.New()
	ta.Prompt = ""
	ta.ShowLineNumbers = true
	ta.SetValue(file)

	fl := list.New(getFileTree(cfg.Root), list.NewDefaultDelegate(), 0, 0)
	fl.SetShowHelp(false)
	fl.Title = "Files"

	sb := statusbar.New(
		statusbar.ColorConfig{
			Foreground: lipgloss.AdaptiveColor{Dark: "#ffffff", Light: "#ffffff"},
			Background: lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F25D94"},
		},
		statusbar.ColorConfig{
			Foreground: lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"},
			Background: lipgloss.AdaptiveColor{Light: "#3c3836", Dark: "#3c3836"},
		},
		statusbar.ColorConfig{
			Foreground: lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"},
			Background: lipgloss.AdaptiveColor{Light: "#A550DF", Dark: "#A550DF"},
		},
		statusbar.ColorConfig{
			Foreground: lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"},
			Background: lipgloss.AdaptiveColor{Light: "#6124DF", Dark: "#6124DF"},
		},
	)

	sb.SetContent(cfg.LastFile, cfg.Root, "edit", "normal")
	return Model{
		config:    cfg,
		textarea:  ta,
		filelist:  fl,
		statusbar: sb,
		height:    0,
		width:     0,
		keymap:    keymaps.GetNormalKeyMaps(),
		help:      help.New(),
		contents:  file,
		state:     initalizing,
	}
}

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height-4, msg.Width
		m.filelist.SetSize(m.width, m.height)
		m.textarea.SetWidth(m.width - 20)
		m.textarea.SetHeight(m.height)

		m.statusbar.SetSize(m.width)

		if m.state == initalizing {
			m = m.changeState(edit)
		}
		return m, nil
	default:
		switch m.state {
		case edit:
			m, cmd = m.updateEdit(msg)
			cmds = append(cmds, cmd)
		case files:
			m, cmd = m.updateFiles(msg)
			cmds = append(cmds, cmd)
		}
	}
	m.statusbar.SetContent(m.config.LastFile, m.config.Root, m.state.String(), m.textarea.Mode.String())
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) updateEdit(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Quit):
			if m.textarea.InNormalMode() {
				return m, tea.Quit
			}
		case key.Matches(msg, m.keymap.ToggleFiles):
			m = m.changeState(files)
			m.textarea.ToNormalMode()
		}
	}
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) updateFiles(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.ToggleFiles):
			m = m.changeState(edit)
		}
	}
	m.filelist, cmd = m.filelist.Update(msg)
	return m, cmd
}

func (m Model) changeState(targetState state) Model {
	switch targetState {
	case files:
		m.state = files
		m.textarea.Blur()
	case edit:
		m.state = edit
		m.textarea.Focus()
	case tooSmall:
		m.state = tooSmall
	}
	return m
}
func (m Model) View() string {
	if m.height < 70 || m.width < 50 {
		return m.tooSmallView()
	}

	var content string
	var help string

	switch m.state {
	case edit:
		content, help = m.editView()
	case files:
		content, help = m.filesView()
	case initalizing:
		return "initializing..."
	}
	appShell := lipgloss.JoinVertical(lipgloss.Top, content, help, m.statusbar.View())
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, appShell)
}

var (
	activeStyle   = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder())
	inactiveStyle = lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder())
	filesStyle    = lipgloss.NewStyle().PaddingRight(5)
)

func (m Model) editView() (string, string) {
	help := m.help.ShortHelpView(m.textarea.ShortHelp())
	innerContent := lipgloss.JoinHorizontal(lipgloss.Left, inactiveStyle.Render(filesStyle.Render(m.filelist.View())), activeStyle.Render(m.textarea.View()))
	return innerContent, help
}
func (m Model) filesView() (string, string) {
	help := m.help.ShortHelpView(m.filelist.ShortHelp())
	innerContent := lipgloss.JoinHorizontal(lipgloss.Left, activeStyle.Render(filesStyle.Render(m.filelist.View())), inactiveStyle.Render(m.textarea.View()))
	return innerContent, help
}

func (m Model) tooSmallView() string {
	return fmt.Sprintf("Window too small: H -> %d W -> %d", m.height, m.width)
}
