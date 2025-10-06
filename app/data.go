package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	github "github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"
)

func githubClient(ctx context.Context) *github.Client {
	tok := getenv("GITHUB_TOKEN")
	if tok == "" {
		return github.NewClient(nil)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok})
	return github.NewClient(oauth2.NewClient(ctx, ts))
}

func fetchRequested(m Model) tea.Cmd {
	return func() tea.Msg {
		q := "is:pr is:open review-requested:@me"
		res, _, err := m.cli.Search.Issues(m.ctx, q, &github.SearchOptions{Sort: "updated", Order: "desc"})
		if err != nil {
			return errMsg(err)
		}
		items := make([]list.Item, 0, len(res.Issues))
		for _, it := range res.Issues {
			repoFull := repoFromIssue(it)
			items = append(items, prItem{
				title:   safe(it.GetTitle()),
				repo:    repoFull,
				number:  it.GetNumber(),
				author:  it.GetUser().GetLogin(),
				updated: it.GetUpdatedAt().Time,
				htmlURL: it.GetHTMLURL(),
			})
		}
		return loadedRequestedMsg(items)
	}
}

func fetchMine(m Model) tea.Cmd {
	return func() tea.Msg {
		q := "is:pr is:open author:@me"
		res, _, err := m.cli.Search.Issues(m.ctx, q, &github.SearchOptions{Sort: "updated", Order: "desc"})
		if err != nil {
			return errMsg(err)
		}
		items := make([]list.Item, 0, len(res.Issues))
		for _, it := range res.Issues {
			repoFull := repoFromIssue(it)
			items = append(items, prItem{
				title:   safe(it.GetTitle()),
				repo:    repoFull,
				number:  it.GetNumber(),
				author:  it.GetUser().GetLogin(),
				updated: it.GetUpdatedAt().Time,
				htmlURL: it.GetHTMLURL(),
			})
		}
		return loadedMineMsg(items)
	}
}

func fetchRuns(m Model) tea.Cmd {
	if m.Owner == "" || m.Repo == "" {
		return nil
	}
	return func() tea.Msg {
		opts := &github.ListWorkflowRunsOptions{ListOptions: github.ListOptions{PerPage: 50}}
		res, _, err := m.cli.Actions.ListRepositoryWorkflowRuns(m.ctx, m.Owner, m.Repo, opts)
		if err != nil {
			return errMsg(err)
		}
		items := make([]list.Item, 0, len(res.WorkflowRuns))
		for _, run := range res.WorkflowRuns {
			name := run.GetName()
			if name == "" {
				name = fmt.Sprintf("run #%d", run.GetRunNumber())
			}
			items = append(items, runItem{
				name:    name,
				branch:  run.GetHeadBranch(),
				status:  run.GetStatus(),
				concl:   run.GetConclusion(),
				updated: run.GetUpdatedAt().Time,
				htmlURL: safe(run.GetHTMLURL()),
				id:      run.GetID(),
			})
		}
		return loadedRunsMsg(items)
	}
}

func fetchPRDetails(m Model, p prItem) tea.Cmd {
	return func() tea.Msg {
		owner, repo := splitRepo(p.repo)
		if owner == "" || repo == "" {
			return detailsMsg("(cannot determine repo)")
		}
		pr, _, err := m.cli.PullRequests.Get(m.ctx, owner, repo, p.number)
		if err != nil {
			return detailsMsg(fmt.Sprintf("error loading PR: %v", err))
		}
		files, _, _ := m.cli.PullRequests.ListFiles(m.ctx, owner, repo, p.number, &github.ListOptions{PerPage: 100})
		revs, _, _ := m.cli.PullRequests.ListReviews(m.ctx, owner, repo, p.number, &github.ListOptions{PerPage: 100})

		var b strings.Builder
		fmt.Fprintf(&b, "# %s\n", pr.GetTitle())
		fmt.Fprintf(&b, "PR %s#%d by @%s  base=%s  head=%s\n", p.repo, p.number, pr.GetUser().GetLogin(), pr.GetBase().GetRef(), pr.GetHead().GetRef())
		fmt.Fprintf(&b, "State: %s  Updated: %s  Draft: %v\n", pr.GetState(), timeAgo(pr.GetUpdatedAt().Time), pr.GetDraft())

		if len(revs) > 0 {
			fmt.Fprintf(&b, "\nReviews (latest states not de-duped):\n")
			for _, r := range revs {
				fmt.Fprintf(&b, "  - @%s: %s\n", r.GetUser().GetLogin(), r.GetState())
			}
		}

		if len(files) > 0 {
			fmt.Fprintf(&b, "\nFiles (%d):\n", len(files))
			max := 50
			if len(files) < max {
				max = len(files)
			}
			for i := 0; i < max; i++ {
				f := files[i]
				fmt.Fprintf(&b, "  - %s (+%d/-%d)\n", f.GetFilename(), f.GetAdditions(), f.GetDeletions())
			}
			if len(files) > max {
				fmt.Fprintf(&b, "  ... and %d more\n", len(files)-max)
			}
		}

		body := strings.TrimSpace(pr.GetBody())
		if body != "" {
			fmt.Fprintf(&b, "\nDescription:\n%s\n", body)
		}
		return detailsMsg(b.String())
	}
}

func fetchRunDetails(m Model, r runItem) tea.Cmd {
	return func() tea.Msg {
		if m.Owner == "" || m.Repo == "" {
			return detailsMsg("(no repo detected)")
		}
		jobs, _, err := m.cli.Actions.ListWorkflowJobs(m.ctx, m.Owner, m.Repo, r.id, &github.ListWorkflowJobsOptions{Filter: "all", ListOptions: github.ListOptions{PerPage: 100}})
		if err != nil {
			return detailsMsg(fmt.Sprintf("error loading jobs: %v", err))
		}
		var b strings.Builder
		fmt.Fprintf(&b, "# %s\n", r.name)
		fmt.Fprintf(&b, "Branch: %s  Status: %s  Conclusion: %s  Updated: %s\n", r.branch, r.status, r.concl, timeAgo(r.updated))
		fmt.Fprintf(&b, "\nJobs (%d):\n", len(jobs.Jobs))
		for _, j := range jobs.Jobs {
			st := j.GetStatus()
			if j.GetConclusion() != "" {
				st = fmt.Sprintf("%s/%s", st, j.GetConclusion())
			}
			fmt.Fprintf(&b, "  - %s [%s]\n", j.GetName(), st)
		}
		return detailsMsg(b.String())
	}
}
