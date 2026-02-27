package git

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/agnishcc/worktree-tui/internal/types"
)

// run executes a git command in the current working directory.
// On failure the returned error includes git's stderr output.
func run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil && stderr.Len() > 0 {
		return strings.TrimSpace(string(out)), fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), err
}

// runInDir executes a git command with the given directory as CWD.
// On failure the returned error includes git's stderr output.
func runInDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil && stderr.Len() > 0 {
		return strings.TrimSpace(string(out)), fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), err
}

// IsGitRepo returns true if the current directory is inside a git repository.
func IsGitRepo() bool {
	_, err := run("rev-parse", "--git-dir")
	return err == nil
}

// HasCommits returns true if the repo at repoRoot has at least one commit.
func HasCommits(repoRoot string) bool {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoRoot
	return cmd.Run() == nil
}

// InitRepo runs git init in the current directory.
func InitRepo() error {
	return exec.Command("git", "init").Run()
}

// GetRepoRoot returns the absolute path to the repository root.
func GetRepoRoot() (string, error) {
	return run("rev-parse", "--show-toplevel")
}

// GetRepoInfo returns the repo's base name and the current branch name.
func GetRepoInfo() (name, branch string, err error) {
	root, err := run("rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", err
	}
	name = filepath.Base(root)
	branch, err = run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		branch = "unknown"
		err = nil
	}
	return name, branch, nil
}

// getDefaultBranch detects the repo's default branch (main/master).
func getDefaultBranch() string {
	if out, err := run("symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		// "origin/main" → "main"
		if parts := strings.SplitN(out, "/", 2); len(parts) == 2 {
			return parts[1]
		}
	}
	if _, err := run("rev-parse", "--verify", "main"); err == nil {
		return "main"
	}
	return "master"
}

// GetBranchStatus returns how many commits a branch is ahead/behind the default
// branch, and whether it has been merged.
func GetBranchStatus(branch string) (ahead, behind int, merged bool, err error) {
	def := getDefaultBranch()
	if branch == def {
		return 0, 0, false, nil
	}

	if out, e := run("rev-list", "--count", def+".."+branch); e == nil {
		ahead, _ = strconv.Atoi(out)
	}
	if out, e := run("rev-list", "--count", branch+".."+def); e == nil {
		behind, _ = strconv.Atoi(out)
	}
	if out, e := run("branch", "--merged", def); e == nil {
		for _, line := range strings.Split(out, "\n") {
			if strings.TrimSpace(strings.TrimPrefix(line, "* ")) == branch {
				merged = true
				break
			}
		}
	}
	return ahead, behind, merged, nil
}

// ListWorktrees returns all worktrees for the current repo, enriched with
// user metadata and branch status.
func ListWorktrees() ([]types.Worktree, error) {
	out, err := run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	root, _ := GetRepoRoot()
	meta, _ := readMeta(root)

	var worktrees []types.Worktree
	for i, block := range strings.Split(strings.TrimSpace(out), "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		wt := types.Worktree{IsMain: i == 0}
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				wt.Path = strings.TrimPrefix(line, "worktree ")
			case strings.HasPrefix(line, "branch "):
				wt.Branch = strings.TrimPrefix(line, "branch refs/heads/")
				wt.Name = wt.Branch // default display name
			case line == "detached":
				wt.Branch = "(detached)"
				wt.Name = "(detached)"
			case line == "bare":
				wt.Branch = "(bare)"
				wt.Name = "(bare)"
			}
		}
		if wt.Name == "" {
			wt.Name = filepath.Base(wt.Path)
		}

		// Overlay user metadata (name, description, createdFrom).
		if m, ok := meta[wt.Branch]; ok {
			if m.Name != "" {
				wt.Name = m.Name
			}
			wt.Description = m.Description
			wt.CreatedFrom = m.CreatedFrom
		}

		// Branch status and detail extras (skip for main worktree).
		if !wt.IsMain {
			wt.Ahead, wt.Behind, wt.IsMerged, _ = GetBranchStatus(wt.Branch)
		}
		wt.HeadSHA, _ = GetHeadSHA(wt.Path)
		wt.StatusChanged, wt.StatusUntracked, _ = GetWorktreeStatus(wt.Path)

		if updated, e := runInDir(wt.Path, "log", "-1", "--format=%cr"); e == nil && updated != "" {
			wt.UpdatedAt = updated
		} else {
			wt.UpdatedAt = "never"
		}

		wt.Commits, _ = GetCommits(wt.Path)
		worktrees = append(worktrees, wt)
	}
	return worktrees, nil
}

// GetCommits returns the last 10 commits for the worktree at path.
func GetCommits(worktreePath string) ([]types.Commit, error) {
	out, err := runInDir(worktreePath, "log", "-10", "--format=%h|%s|%cr")
	if err != nil || out == "" {
		return nil, err
	}
	var commits []types.Commit
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		commits = append(commits, types.Commit{
			Hash:    parts[0],
			Message: parts[1],
			RelTime: parts[2],
		})
	}
	return commits, nil
}

// AddWorktree creates a new worktree with a new branch at wtPath.
func AddWorktree(branch, wtPath string) error {
	_, err := run("worktree", "add", "-b", branch, wtPath, "HEAD")
	return err
}

// RemoveWorktree force-removes the worktree at path.
func RemoveWorktree(path string) error {
	_, err := run("worktree", "remove", "--force", path)
	return err
}

// RenameBranch renames a branch in the current repository.
func RenameBranch(oldName, newName string) error {
	_, err := run("branch", "-m", oldName, newName)
	return err
}

// ── Repo-global info ──────────────────────────────────────────────────────────

// GetDefaultBranch exports the default-branch detection logic.
func GetDefaultBranch() string { return getDefaultBranch() }

// GetRemoteURL returns the origin remote URL shortened to "host/org/repo".
func GetRemoteURL() (string, error) {
	url, err := run("remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	return shortenURL(url), nil
}

func shortenURL(url string) string {
	for _, pfx := range []string{"https://", "http://", "git@", "ssh://"} {
		url = strings.TrimPrefix(url, pfx)
	}
	url = strings.Replace(url, ":", "/", 1) // git@github.com:org/repo → github.com/org/repo
	url = strings.TrimSuffix(url, ".git")
	return url
}

// GetStashCount returns the number of stash entries.
func GetStashCount() (int, error) {
	out, err := run("stash", "list")
	if err != nil || strings.TrimSpace(out) == "" {
		return 0, nil
	}
	return len(strings.Split(strings.TrimSpace(out), "\n")), nil
}

// GetFetchedAgo returns a human-readable relative time since the last fetch,
// or ("", nil) if FETCH_HEAD does not exist.
func GetFetchedAgo() (string, error) {
	root, err := GetRepoRoot()
	if err != nil {
		return "", err
	}
	info, err := os.Stat(filepath.Join(root, ".git", "FETCH_HEAD"))
	if err != nil {
		return "", nil // not an error — just hasn't been fetched yet
	}
	return fmtDuration(time.Since(info.ModTime())), nil
}

func fmtDuration(d time.Duration) string {
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

// ── Per-worktree detail extras ────────────────────────────────────────────────

// GetHeadSHA returns the short SHA of HEAD for the worktree at path.
func GetHeadSHA(worktreePath string) (string, error) {
	return runInDir(worktreePath, "rev-parse", "--short", "HEAD")
}

// GetWorktreeStatus returns counts of changed and untracked files.
func GetWorktreeStatus(worktreePath string) (changed, untracked int, err error) {
	out, err := runInDir(worktreePath, "status", "--porcelain")
	if err != nil {
		return 0, 0, err
	}
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 {
			continue
		}
		if strings.HasPrefix(line, "??") {
			untracked++
		} else {
			changed++
		}
	}
	return changed, untracked, nil
}

// ── PR badge (gh CLI) ─────────────────────────────────────────────────────────

// IsGHAvailable returns true if the gh CLI binary is on PATH.
func IsGHAvailable() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// GetPRInfo fetches PR state/number/URL for branch via gh. Returns nil if no
// PR exists, gh is unavailable, or the call fails.
func GetPRInfo(branch string) (*types.PRInfo, error) {
	out, err := exec.Command("gh", "pr", "view", branch,
		"--json", "state,number,url").Output()
	if err != nil {
		return nil, nil // no PR or gh not available
	}
	var v struct {
		State  string `json:"state"`
		Number int    `json:"number"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return nil, nil
	}
	return &types.PRInfo{State: v.State, Number: v.Number, URL: v.URL}, nil
}

// GetCommitDetail fetches full commit data (subject, body, files changed, diff)
// for the given short or full SHA in the worktree at worktreePath.
func GetCommitDetail(worktreePath, sha string) (*types.CommitDetail, error) {
	shortHash, _ := runInDir(worktreePath, "show", sha, "--no-patch", "--pretty=format:%h")
	subject, _ := runInDir(worktreePath, "show", sha, "--no-patch", "--pretty=format:%s")
	body, _ := runInDir(worktreePath, "show", sha, "--no-patch", "--pretty=format:%b")
	relTime, _ := runInDir(worktreePath, "show", sha, "--no-patch", "--pretty=format:%cr")

	// --pretty=format: (empty) suppresses the commit header so we get just the list.
	filesOut, _ := runInDir(worktreePath, "show", sha, "--name-status", "--no-patch", "--pretty=format:")
	diffOut, _ := runInDir(worktreePath, "show", sha, "--patch", "--no-color", "--pretty=format:")

	detail := &types.CommitDetail{
		ShortHash: shortHash,
		Subject:   subject,
		Body:      strings.TrimRight(body, "\r\n"),
		RelTime:   relTime,
		Loaded:    true,
	}

	// Parse file status list.
	for _, line := range strings.Split(filesOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		// Rename/copy status comes as "R090", "C100", etc. — normalise to single letter.
		if len(status) > 1 {
			status = string(status[0])
		}
		path := parts[len(parts)-1] // for renames the new path is last
		detail.Files = append(detail.Files, types.CommitFile{Status: status, Path: path})
	}

	// Parse diff output line by line.
	for _, line := range strings.Split(diffOut, "\n") {
		var dt string
		switch {
		case strings.HasPrefix(line, "diff --git"):
			dt = "diff"
		case strings.HasPrefix(line, "index "),
			strings.HasPrefix(line, "new file"),
			strings.HasPrefix(line, "deleted file"),
			strings.HasPrefix(line, "--- "),
			strings.HasPrefix(line, "+++ "):
			dt = "meta"
		case strings.HasPrefix(line, "@@"):
			dt = "@@"
		case strings.HasPrefix(line, "+"):
			dt = "+"
		case strings.HasPrefix(line, "-"):
			dt = "-"
		default:
			dt = " "
		}
		detail.Diff = append(detail.Diff, types.DiffLine{Type: dt, Content: line})
	}

	return detail, nil
}

// SaveWorktreeMeta stores user-defined metadata for a worktree.
// It captures the current HEAD SHA as the createdFrom commit.
func SaveWorktreeMeta(branch, name, description string) error {
	root, err := GetRepoRoot()
	if err != nil {
		return err
	}
	meta, _ := readMeta(root)
	if meta == nil {
		meta = make(map[string]WorktreeMeta)
	}
	head, _ := run("rev-parse", "--short", "HEAD")
	meta[branch] = WorktreeMeta{
		Name:        name,
		Description: description,
		CreatedFrom: head,
	}
	return writeMeta(root, meta)
}

// DeleteWorktreeMeta removes the metadata entry for a branch.
func DeleteWorktreeMeta(branch string) error {
	root, err := GetRepoRoot()
	if err != nil {
		return err
	}
	meta, _ := readMeta(root)
	if meta == nil {
		return nil
	}
	delete(meta, branch)
	return writeMeta(root, meta)
}

// --- Metadata persistence ---

// WorktreeMeta is the user-defined metadata persisted per branch.
type WorktreeMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedFrom string `json:"createdFrom"`
}

func metaFilePath(repoRoot string) string {
	return filepath.Join(repoRoot, ".git", "worktree-tui", "meta.json")
}

func readMeta(repoRoot string) (map[string]WorktreeMeta, error) {
	data, err := os.ReadFile(metaFilePath(repoRoot))
	if err != nil {
		return make(map[string]WorktreeMeta), nil
	}
	var m map[string]WorktreeMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return make(map[string]WorktreeMeta), nil
	}
	return m, nil
}

func writeMeta(repoRoot string, meta map[string]WorktreeMeta) error {
	p := metaFilePath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// --- Shell integration ---

const cdTempFile = "/tmp/.wt_cd_path"

func markerPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "worktree-tui", "integrated"), nil
}

// IsShellIntegrated returns true if the shell setup prompt has already been shown.
func IsShellIntegrated() bool {
	p, err := markerPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// MarkShellIntegrated writes the marker so the setup prompt is not shown again.
func MarkShellIntegrated() error {
	p, err := markerPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte("1"), 0o644)
}

// SetupShellIntegration appends the wt() wrapper to the user's shell rc file.
func SetupShellIntegration() error {
	shell := os.Getenv("SHELL")
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	var rcFile string
	switch {
	case strings.Contains(shell, "zsh"):
		rcFile = filepath.Join(home, ".zshrc")
	case strings.Contains(shell, "bash"):
		rcFile = filepath.Join(home, ".bashrc")
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
	fn := `
# worktree-tui shell integration
wt() {
  worktree-tui "$@"
  if [ -f /tmp/.wt_cd_path ]; then
    cd "$(cat /tmp/.wt_cd_path)"
    rm /tmp/.wt_cd_path
  fi
}
`
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(fn)
	return err
}

// WriteCDPath writes the target path to the temp file read by the shell wrapper.
func WriteCDPath(path string) error {
	return os.WriteFile(cdTempFile, []byte(path), 0o644)
}
