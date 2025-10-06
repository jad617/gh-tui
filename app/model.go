package app

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	github "github.com/google/go-github/v61/github"
)

const appTitle = "GH TUI - PRs & Actions"

var (
	tabNames = []string{"Requested Reviews", "My PRs", "Actions"}

	// Colors
	colorTitle  = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#B794F4"} // purple highlight
	colorSubtle = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#AAAAAA"}
	colorActive = colorTitle
	colorBorder = lipgloss.Color("#ff9e64") // your orange border

	// Tab row
	tabStyle  = lipgloss.NewStyle().Padding(0, 2).Foreground(colorSubtle)
	activeTab = tabStyle.Copy().Foreground(colorActive).Bold(true)

	// Boxes for panes
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	subtleStyle = lipgloss.NewStyle().Foreground(colorSubtle)
)

type openable interface{ url() string }

type prItem struct {
	title   string
	repo    string
	number  int
	author  string
	updated time.Time
	htmlURL string
}

func (p prItem) Title() string {
	return fmt.Sprintf("%s %s#%d", p.title, subtleStyle.Render(p.repo), p.number)
}

func (p prItem) Description() string {
	return fmt.Sprintf("by %s - updated %s", p.author, timeAgo(p.updated))
}

func (p prItem) FilterValue() string { return p.title }

func (p prItem) url() string { return p.htmlURL }

type runItem struct {
	name    string
	branch  string
	status  string
	concl   string
	updated time.Time
	htmlURL string
	id      int64
}

func (r runItem) Title() string {
	return fmt.Sprintf("%s %s [%s/%s]", r.name, subtleStyle.Render(r.branch), r.status, r.concl)
}

func (r runItem) Description() string { return fmt.Sprintf("updated %s", timeAgo(r.updated)) }

func (r runItem) FilterValue() string { return r.name }

func (r runItem) url() string { return r.htmlURL }

type tab int

const (
	tabRequested tab = iota
	tabMine
	tabActions
)

type Model struct {
	ctx context.Context
	cli *github.Client

	width  int
	height int

	activeTab tab
	lists     []list.Model

	StatusMsg string
	Owner     string
	Repo      string

	ShowDetails bool
	Vp          viewport.Model

	Tailing bool
}

func New() Model {
	ctx := context.Background()
	cli := githubClient(ctx)

	// Each list shows its title inside the box.
	listTitles := []string{"Requested Reviews", "My PRs", "Actions"}
	mkList := func(title string) list.Model {
		l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
		l.Title = title
		l.SetShowHelp(false)
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(true)
		return l
	}

	m := Model{ctx: ctx, cli: cli, activeTab: tabRequested}
	m.lists = []list.Model{
		mkList(listTitles[0]),
		mkList(listTitles[1]),
		mkList(listTitles[2]),
	}
	m.Vp = viewport.New(0, 0)

	if o, r, err := getRepoFromGit(); err == nil {
		m.Owner, m.Repo = o, r
		m.StatusMsg = fmt.Sprintf("Detected repo: %s/%s", o, r)
	} else {
		m.StatusMsg = "Tip: run inside a git repo to enable Actions tab, or set GITHUB_REPOSITORY=owner/repo"
	}
	return m
}
