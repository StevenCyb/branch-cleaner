package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BranchInfo struct {
	Name             string
	NotOnRemote      bool
	HasUniqueCommits bool
	IsBehindMain     bool
	LastCommit       time.Time
}

type item BranchInfo

func (i item) Title() string { return i.Name }
func (i item) Description() string {
	status := []string{}
	if i.NotOnRemote {
		status = append(status, "not on remote")
	}
	if i.HasUniqueCommits {
		status = append(status, "✓ unique commits")
	}
	if i.IsBehindMain {
		status = append(status, "✓ behind main")
	}
	statusStr := strings.Join(status, ", ")
	if statusStr != "" {
		statusStr += ", "
	}
	return fmt.Sprintf("%sLast commit: %s",
		statusStr,
		i.LastCommit.Format("2006-01-02 15:04:05"))
}
func (i item) FilterValue() string { return i.Name }

type model struct {
	list     list.Model
	branches []BranchInfo
	marked   map[int]struct{}
	quitting bool
	// Names of deleted branches for summary output
	deleted []string
}

func InitialModel(branches []BranchInfo) tea.Model {
	items := make([]list.Item, len(branches))
	for i, b := range branches {
		items[i] = item(b)
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Local branches not on remote"
	return model{
		list:     l,
		branches: branches,
		marked:   make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) { //nolint:gocritic // want to switch on type
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case " ":
			idx := m.list.Index()
			if _, ok := m.marked[idx]; ok {
				delete(m.marked, idx)
			} else {
				m.marked[idx] = struct{}{}
			}

		case "enter":
			if len(m.marked) == 0 {
				m.quitting = true
				return m, tea.Quit
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var toDelete []string
			for idx := range m.marked {
				branch := m.branches[idx].Name
				if branch == "main" || branch == "master" {
					continue
				}
				toDelete = append(toDelete, branch)
			}

			deleted := DeleteBranches(ctx, toDelete)

			m.deleted = deleted
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		if len(m.deleted) > 0 {
			var b strings.Builder
			fmt.Fprintf(&b, "Deleted %d branches:\n", len(m.deleted))
			for _, name := range m.deleted {
				fmt.Fprintf(&b, "* %s\n", name)
			}
			return b.String()
		}
		if len(m.marked) == 0 {
			return "No branches to delete.\n"
		}
		return "Deleted marked branches.\n"
	}

	header := fmt.Sprintf("%-3s %-30s %-7s %-7s %-7s %-20s",
		"", "Branch", "Remote", "Unique", "Behind", "Last Commit")
	sep := strings.Repeat("-", 65)

	rows := []string{header, sep}
	cursor := m.list.Index()

	for i, b := range m.branches {
		mark := "[ ]"
		if _, ok := m.marked[i]; ok {
			mark = "[x]"
		}

		remote := "✗"
		if !b.NotOnRemote {
			remote = "✓"
		}
		unique := "✗"
		if b.HasUniqueCommits {
			unique = "✓"
		}
		behind := "✗"
		if b.IsBehindMain {
			behind = "✓"
		}
		last := ""
		if !b.LastCommit.IsZero() {
			last = b.LastCommit.Format("2006-01-02 15:04:05")
		}
		row := fmt.Sprintf("%s %-30s %-7s %-7s %-7s %-20s",
			mark, b.Name, remote, unique, behind, last)
		if i == cursor {
			row = "\x1b[7m" + row + "\x1b[0m"
		}
		rows = append(rows, row)
	}

	help := "\nUse ↑/↓ or k/j to move, <space> to mark, <enter> to delete, q to quit."
	helpStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(help)
	return strings.Join(rows, "\n") + helpStyled
}
