package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func openURL(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", u)
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Start()
}

func getRepoFromGit() (owner, repo string, _ error) {
	if env := getenv("GITHUB_REPOSITORY"); env != "" {
		parts := strings.Split(env, "/")
		if len(parts) == 2 {
			return parts[0], parts[1], nil
		}
	}
	out, err := exec.Command("git", "config", "--get", "remote.origin.url").CombinedOutput()
	if err != nil {
		return "", "", errors.New("not a git repo or no remote.origin.url")
	}
	remote := strings.TrimSpace(string(out))
	re := regexp.MustCompile(`(?i)github\.com[:/]{1,2}([^/]+)/([^/.]+)`)
	m := re.FindStringSubmatch(remote)
	if len(m) == 3 {
		return m[1], m[2], nil
	}
	return "", "", fmt.Errorf("could not parse GitHub repo from remote: %s", remote)
}

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
func safe(s string) string { return strings.ReplaceAll(s, "\n", " ") }

func repoFromIssue(it interface{ GetRepositoryURL() string; GetHTMLURL() string }) string {
	repoURL := it.GetRepositoryURL()
	parts := strings.Split(repoURL, "/repos/")
	if len(parts) == 2 {
		return parts[1]
	}
	h := it.GetHTMLURL()
	if i := strings.Index(h, "://github.com/"); i != -1 {
		return strings.TrimPrefix(h[i+len("://github.com/"):], "/")
	}
	return ""
}

func splitRepo(full string) (string, string) {
	parts := strings.Split(full, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func getenv(k string) string { return os.Getenv(k) }
