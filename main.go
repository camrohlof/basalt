package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	// "path/filepath"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	initalizing state = iota
	viewer
	fileExplorer
)

type mode int

const (
	normal mode = iota
	files
	leader
)

var (
	modelStyle        = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).BorderStyle(lipgloss.HiddenBorder())
	focusedModelStyle = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).BorderStyle(lipgloss.NormalBorder())
)

type keymap = struct {
	editMode, normalMode, openFiles, openViewer, quit, leader, selectFile key.Binding
}

func getNormalKeyMaps() keymap {
	return keymap{
		normalMode: key.NewBinding(key.WithDisabled()),
		quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "exit"),
		),
		leader: key.NewBinding(
			key.WithKeys("space"),
			key.WithHelp("space", "leader"),
		),
		openFiles: key.NewBinding(
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

func getFilesKeyMaps() keymap {
	return keymap{
		quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "exit"),
		),
		openViewer: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "viewer"),
		),
		selectFile: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		openFiles: key.NewBinding(key.WithDisabled()),
	}
}

type model struct {
	viewport    viewport.Model
	filepicker  filepicker.Model
	currentFile string
	contents    string
	keymap      keymap
	help        help.Model
	mode        mode
	err         error
	state       state
}

func (m *model) switchModes(mode mode) {
	switch mode {
	case normal:
		m.mode = normal
		m.state = viewer
		m.keymap = getNormalKeyMaps()
	case files:
		m.mode = files
		m.state = fileExplorer
		m.keymap = getFilesKeyMaps()
	case leader:
		m.mode = leader
	}
}

type errMsg struct{ err error }

type succMsg struct{}

func writeToFile(value string) tea.Cmd {
	return func() tea.Msg {
		err := os.WriteFile("test.md", []byte(value), 0644)
		if err != nil {
			return errMsg{err}
		}
		return succMsg{}
	}
}

type editorFinishedMsg struct{ err error }

func openEditor(fileName string) tea.Cmd {
	editor := "nvim"
	c := exec.Command(editor, fileName)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err}
	})
}

func getFirstFile() string {
	out, _ := os.ReadFile("test.md")
	return string(out)
}

type newFileMsg struct {
	path     string
	contents string
	err      error
}

func newFileSelected(path string) tea.Cmd {
	log.Println(path)
	return func() tea.Msg {
		contents, err := os.ReadFile(path)
		if err != nil {
			log.Println(err.Error())
		}
		return newFileMsg{path, string(contents[:]), err}
	}
}

func initialModel() model {
	file := getFirstFile()
	vp := viewport.New(100, 100)
	vp.SetContent(file)

	fp := filepicker.New()
	fp.AllowedTypes = []string{".md"}

	return model{
		viewport:    vp,
		filepicker:  fp,
		currentFile: "test.md",
		contents:    file,
		keymap:      getNormalKeyMaps(),
		help:        help.New(),
		mode:        normal,
		err:         nil,
		state:       initalizing,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, m.filepicker.Init(), tea.SetWindowTitle("Basalt"))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.editMode):
			return m, openEditor(m.currentFile)
		case key.Matches(msg, m.keymap.openFiles):
			m.switchModes(files)
			m.filepicker, cmd = m.filepicker.Update(msg)
			cmds = append(cmds, cmd)
		case key.Matches(msg, m.keymap.openViewer):
			m.switchModes(normal)
		case key.Matches(msg, m.keymap.selectFile):
		}

	case editorFinishedMsg:
		if msg.err != nil {
			log.Fatalf(msg.err.Error())
		}
		m.contents = getFirstFile()
	case tea.WindowSizeMsg:
		m.viewport.Height = msg.Height - 20
		m.viewport.Width = msg.Width
	case newFileMsg:
		m.contents = msg.contents
		m.currentFile = msg.path
	case errMsg:
		m.err = msg.err
		return m, nil
	}
	if m.state == initalizing {
		m.state = viewer
	}
	m.filepicker, cmd = m.filepicker.Update(msg)
	cmds = append(cmds, cmd)

	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		log.Println(didSelect)
		cmd := newFileSelected(path)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func mainView(m model) string {
	help := m.help.ShortHelpView([]key.Binding{
		m.keymap.quit,
		m.keymap.editMode,
		m.keymap.normalMode,
		m.keymap.leader,
	})
	var s string
	style := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).PaddingLeft(10).PaddingRight(10)
	glamourString, _ := glamour.Render(m.contents, "dark")
	m.viewport.SetContent(glamourString)
	s += m.viewport.View() + "\n\n"
	s += help + "\n"
	s += m.currentFile + "\n"
	return lipgloss.Place(m.viewport.Width, m.viewport.Height, lipgloss.Center, lipgloss.Center, style.Render(s))
}

func filesView(m model) string {
	help := m.help.ShortHelpView([]key.Binding{
		m.keymap.quit,
		m.keymap.normalMode,
	})
	style := lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).PaddingLeft(20).PaddingRight(20).PaddingTop(10).PaddingBottom(10)
	var s string
	s += m.filepicker.View() + "\n\n"
	s += help + "\n"
	s += m.currentFile + "\n"
	return lipgloss.Place(m.viewport.Width, m.viewport.Height, lipgloss.Center, lipgloss.Center, style.Render(s))
}

func (m model) View() string {
	if m.state == initalizing {
		return "loading..."
	}
	switch m.state {
	case viewer:
		return mainView(m)
	case fileExplorer:
		return filesView(m)
	default:
		return "how did this happen?"
	}
}

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
