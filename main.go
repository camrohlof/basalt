package main

import (
	"camrohlof/basalt/internal/utils"
	mainview "camrohlof/basalt/internal/views"
	"fmt"
	"log"
	"os"
	"os/exec"

	// "path/filepath"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	initalizing state = iota
	mainView
)

var (
	modelStyle        = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).BorderStyle(lipgloss.HiddenBorder())
	focusedModelStyle = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).BorderStyle(lipgloss.NormalBorder())
)

type model struct {
	mainview mainview.Model
	config   utils.Config
	state    state
	err      error
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
	editor := "zed"
	c := exec.Command(editor, fileName)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err}
	})
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

func initialModel(cfg utils.Config) model {

	return model{
		mainview: mainview.New(cfg),
		config:   cfg,
		err:      nil,
		state:    initalizing,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.EnterAltScreen, tea.SetWindowTitle("Basalt"))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.state == initalizing {
		m.state = mainView
	}
	m.mainview, cmd = m.mainview.Update(msg)
	return m, cmd
}

func (m model) View() string {
	switch m.state {
	case initalizing:
		return "loading..."
	case mainView:
		return m.mainview.View()
	}
	return "How did this happen?"
}

func main() {
	cfg := utils.ReadFromConfig()
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
