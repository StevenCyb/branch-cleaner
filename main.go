package main

import (
	"log"
	"os"
	"path/filepath"

	ui "github.com/StevenCyb/branch-cleaner/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var path string
	if len(os.Args) > 1 {
		path = os.Args[1]
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		path = cwd
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}

	branches, err := ui.GetBranches(abs)
	if err != nil {
		log.Fatal(err)
	}

	if len(branches) == 0 {
		log.Print("No branches to delete.")
		return
	}

	p := tea.NewProgram(ui.InitialModel(branches))
	if _, err := p.Run(); err != nil {
		log.Print("Error:", err)
		os.Exit(1)
	}
}
