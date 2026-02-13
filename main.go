package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	// Prompt and optionally run `git fetch --all --prune` in the repo dir
	fmt.Print("Run 'git fetch --all --prune' to update remotes and prune stale branches? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	resp = strings.TrimSpace(strings.ToLower(resp))
	if resp == "" || resp == "y" || resp == "yes" {
		cmd := exec.Command("git", "fetch", "--all", "--prune")
		cmd.Dir = abs
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("git fetch --all --prune failed: %v", err)
		}
	}

	branches, err := ui.GetBranches(abs)
	if err != nil {
		log.Fatal(err)
	}

	if len(branches) == 0 {
		log.Print("No branches to delete.")
		return
	}

	p := tea.NewProgram(ui.InitialModel(abs, branches))
	if _, err := p.Run(); err != nil {
		log.Print("Error:", err)
		os.Exit(1)
	}
}
