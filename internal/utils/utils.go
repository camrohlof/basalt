package utils

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Root     string
	LastFile string
}

type succMsg struct{ nextCmd tea.Cmd }

func ReadFromConfig() Config {
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
	return cfg
}

func WriteToConfig(currentFile string) tea.Cmd {
	return func() tea.Msg {
		cfg := Config{
			Root:     ".",
			LastFile: currentFile,
		}
		cfgFile, err := toml.Marshal(cfg)
		if err != nil {
			log.Fatal(err)
		}
		err = os.WriteFile("config.toml", cfgFile, 0644)
		if err != nil {
			log.Fatal(err)
		}
		return succMsg{nextCmd: tea.Quit}
	}
}
