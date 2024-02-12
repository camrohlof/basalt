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

	"github.com/pelletier/go-toml/v2"
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

type Config struct {
	Root     string
	LastFile string
}

func (m model) writeToConfig() tea.Cmd {
	currentFile := m.currentFile
	log.Println(currentFile)
	return func() tea.Msg {
		cfg := Config{
			Root:     ".",
			LastFile: currentFile,
		}
		log.Println(cfg.LastFile)
		cfgFile, err := toml.Marshal(cfg)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(string(cfgFile))
		err = os.WriteFile("config.toml", cfgFile, 0644)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("everthing worked")
		return succMsg{nextCmd: tea.Quit}
	}
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

type succMsg struct{ nextCmd tea.Cmd }

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

func getFirstFile(lastFile string) string {
	out, _ := os.ReadFile(lastFile)
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

func initialModel(cfg Config) model {
	file := getFirstFile(cfg.LastFile)
	vp := viewport.New(100, 100)
	vp.SetContent(file)

	fp := filepicker.New()
	fp.AllowedTypes = []string{".md"}

	return model{
		viewport:    vp,
		filepicker:  fp,
		currentFile: cfg.LastFile,
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
			return m, m.writeToConfig()
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
		m.contents = getFirstFile(m.currentFile)
	case tea.WindowSizeMsg:
		m.viewport.Height = msg.Height - 20
		m.viewport.Width = msg.Width
	case newFileMsg:
		m.contents = msg.contents
		m.currentFile = msg.path
	case errMsg:
		m.err = msg.err
		return m, nil
	case succMsg:
		if msg.nextCmd != nil {
			return m, msg.nextCmd
		}
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
	var style lipgloss.Style
	var fullView string
	if m.state == viewer {
		style = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).PaddingLeft(10).PaddingRight(10).PaddingTop(1).MarginTop(5)
		glamourString, _ := glamour.Render(m.contents, "dark")
		m.viewport.SetContent(glamourString)
		s += m.viewport.View() + "\n\n"
		fullView = lipgloss.JoinHorizontal(lipgloss.Top, filesView(m), style.Render(s))
	} else {
		style = lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).PaddingLeft(10).PaddingRight(10).PaddingTop(1).MarginTop(5)
		glamourString, _ := glamour.Render(m.contents, "dark")
		m.viewport.SetContent(glamourString)
		s += m.viewport.View() + "\n\n"
		fullView = lipgloss.JoinHorizontal(lipgloss.Top, filesView(m), style.Render(s))
	}
	//return lipgloss.Place(m.viewport.Width, m.viewport.Height, lipgloss.Center, lipgloss.Center, fullView)
	return fmt.Sprintf("%s \n %s \n", fullView, help)
}

func filesView(m model) string {
	var style lipgloss.Style
	var s string
	if m.state == fileExplorer {
		style = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).PaddingLeft(5).PaddingRight(5).PaddingTop(3).PaddingBottom(10).MarginTop(5)
		s += m.filepicker.View() + "\n\n"
		s += fmt.Sprintf("Current file: %s \n", m.currentFile)
	} else {
		style = lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).PaddingLeft(5).PaddingRight(5).PaddingTop(3).PaddingBottom(10).MarginTop(5)
		s += m.filepicker.View() + "\n\n"
		s += fmt.Sprintf("Current file: %s \n", m.currentFile)
	}
	//return lipgloss.Place(m.viewport.Width, m.viewport.Height, lipgloss.Center, lipgloss.Center, style.Render(s))
	return style.Render(s)
}

func (m model) View() string {
	if m.state == initalizing {
		return "loading..."
	}
	return mainView(m)
}

func main() {
	var cfg Config
	cfgFile, err := os.ReadFile("config.toml")
	if err != nil {
		cfg = Config{
			Root:     ".",
			LastFile: "new.md",
		}
	} else {
		err = toml.Unmarshal([]byte(cfgFile), &cfg)
		if err != nil {
			log.Fatal(err)
		}
	}

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	p := tea.NewProgram(initialModel(cfg))

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

}
