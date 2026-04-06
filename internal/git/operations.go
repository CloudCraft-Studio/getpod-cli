package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner executes git operations on local repositories.
// Uses exec.Command — no external dependencies required.
type Runner struct {
	// BasePath is the root directory where repos are cloned.
	// Defaults to ~/.getpod/repos if empty.
	BasePath string
}

// NewRunner creates a Runner with the given base path.
// If basePath is empty, it defaults to ~/.getpod/repos.
func NewRunner(basePath string) *Runner {
	if basePath == "" {
		home, _ := os.UserHomeDir()
		basePath = filepath.Join(home, ".getpod", "repos")
	}
	return &Runner{BasePath: basePath}
}

// RepoPath returns the full local path for a repo name.
func (r *Runner) RepoPath(repoName string) string {
	return filepath.Join(r.BasePath, repoName)
}

// CloneOrPull clones the repo if it doesn't exist locally, or pulls if it does.
func (r *Runner) CloneOrPull(ctx context.Context, cloneURL, repoName string) error {
	dir := r.RepoPath(repoName)

	if isGitRepo(dir) {
		_, err := r.run(ctx, dir, "pull", "--ff-only")
		if err != nil {
			return fmt.Errorf("git pull in %s: %w", repoName, err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return fmt.Errorf("creating parent dir: %w", err)
	}

	_, err := r.run(ctx, r.BasePath, "clone", cloneURL, repoName)
	if err != nil {
		return fmt.Errorf("git clone %s: %w", repoName, err)
	}
	return nil
}

// CreateBranch creates and checks out a new branch in the given repo.
// Branch name follows conventional format: feature/{issueKey}.
func (r *Runner) CreateBranch(ctx context.Context, repoName, branchName string) error {
	dir := r.RepoPath(repoName)
	if !isGitRepo(dir) {
		return fmt.Errorf("repo %q not found at %s", repoName, dir)
	}

	// Check if branch already exists
	current, _ := r.CurrentBranch(ctx, repoName)
	if current == branchName {
		return nil // already on the target branch
	}

	_, err := r.run(ctx, dir, "checkout", "-b", branchName)
	if err != nil {
		// Branch might exist already — try switching
		_, err2 := r.run(ctx, dir, "checkout", branchName)
		if err2 != nil {
			return fmt.Errorf("creating branch %q in %s: %w", branchName, repoName, err)
		}
	}
	return nil
}

// Status returns a list of modified/untracked files in the repo.
func (r *Runner) Status(ctx context.Context, repoName string) ([]string, error) {
	dir := r.RepoPath(repoName)
	if !isGitRepo(dir) {
		return nil, fmt.Errorf("repo %q not found at %s", repoName, dir)
	}

	out, err := r.run(ctx, dir, "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git status in %s: %w", repoName, err)
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// Commit stages all changes and commits with the given message.
func (r *Runner) Commit(ctx context.Context, repoName, message string) error {
	dir := r.RepoPath(repoName)
	if !isGitRepo(dir) {
		return fmt.Errorf("repo %q not found at %s", repoName, dir)
	}

	if _, err := r.run(ctx, dir, "add", "-A"); err != nil {
		return fmt.Errorf("git add in %s: %w", repoName, err)
	}

	if _, err := r.run(ctx, dir, "commit", "-m", message); err != nil {
		return fmt.Errorf("git commit in %s: %w", repoName, err)
	}
	return nil
}

// Push pushes the current branch to origin.
func (r *Runner) Push(ctx context.Context, repoName string) error {
	dir := r.RepoPath(repoName)
	if !isGitRepo(dir) {
		return fmt.Errorf("repo %q not found at %s", repoName, dir)
	}

	branch, err := r.CurrentBranch(ctx, repoName)
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	if _, err := r.run(ctx, dir, "push", "-u", "origin", branch); err != nil {
		return fmt.Errorf("git push in %s: %w", repoName, err)
	}
	return nil
}

// CurrentBranch returns the name of the current branch.
func (r *Runner) CurrentBranch(ctx context.Context, repoName string) (string, error) {
	dir := r.RepoPath(repoName)
	if !isGitRepo(dir) {
		return "", fmt.Errorf("repo %q not found at %s", repoName, dir)
	}

	out, err := r.run(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting branch in %s: %w", repoName, err)
	}
	return strings.TrimSpace(out), nil
}

// BranchNameForIssue generates a conventional branch name from an issue key.
func BranchNameForIssue(issueKey string) string {
	return "feature/" + strings.ToLower(issueKey)
}

// SuggestBaseBranch suggests the PR base branch based on the environment.
func SuggestBaseBranch(environment string) string {
	switch strings.ToLower(environment) {
	case "qa", "dev", "development":
		return "develop"
	case "stg", "staging":
		return "release"
	case "prod", "production":
		return "main"
	default:
		return "main"
	}
}

// CommitMessage builds a conventional commit message.
func CommitMessage(commitType, scope, description string) string {
	if scope != "" {
		return fmt.Sprintf("%s(%s): %s", commitType, scope, description)
	}
	return fmt.Sprintf("%s: %s", commitType, description)
}

// run executes a git command and returns stdout.
func (r *Runner) run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("%s", errMsg)
	}
	return stdout.String(), nil
}

// isGitRepo checks if a directory contains a .git directory.
func isGitRepo(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}
