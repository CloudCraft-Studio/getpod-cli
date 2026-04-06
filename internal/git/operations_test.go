package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initBareRepo creates a bare repo that can be used as an origin for clone.
func initBareRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bare := filepath.Join(dir, "origin.git")
	cmd := exec.Command("git", "init", "--bare", bare)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	return bare
}

// initLocalRepo creates a local repo with one commit.
func initLocalRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")

	f := filepath.Join(dir, "README.md")
	if err := os.WriteFile(f, []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-m", "initial commit")
}

func TestBranchNameForIssue(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"LULO-1234", "feature/lulo-1234"},
		{"GPOD-118", "feature/gpod-118"},
		{"fix-123", "feature/fix-123"},
	}
	for _, tt := range tests {
		got := BranchNameForIssue(tt.key)
		if got != tt.want {
			t.Errorf("BranchNameForIssue(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestSuggestBaseBranch(t *testing.T) {
	tests := []struct {
		env  string
		want string
	}{
		{"qa", "develop"},
		{"dev", "develop"},
		{"stg", "release"},
		{"staging", "release"},
		{"prod", "main"},
		{"production", "main"},
		{"unknown", "main"},
		{"", "main"},
	}
	for _, tt := range tests {
		got := SuggestBaseBranch(tt.env)
		if got != tt.want {
			t.Errorf("SuggestBaseBranch(%q) = %q, want %q", tt.env, got, tt.want)
		}
	}
}

func TestCommitMessage(t *testing.T) {
	tests := []struct {
		typ, scope, desc string
		want             string
	}{
		{"feat", "auth", "add login", "feat(auth): add login"},
		{"fix", "", "typo in readme", "fix: typo in readme"},
	}
	for _, tt := range tests {
		got := CommitMessage(tt.typ, tt.scope, tt.desc)
		if got != tt.want {
			t.Errorf("CommitMessage(%q,%q,%q) = %q, want %q", tt.typ, tt.scope, tt.desc, got, tt.want)
		}
	}
}

func TestCurrentBranch(t *testing.T) {
	dir := t.TempDir()
	repoName := "test-repo"
	repoDir := filepath.Join(dir, repoName)
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	initLocalRepo(t, repoDir)

	r := NewRunner(dir)
	ctx := context.Background()

	branch, err := r.CurrentBranch(ctx, repoName)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	// Could be "main" or "master" depending on git config
	if branch != "main" && branch != "master" {
		t.Errorf("expected main or master, got %q", branch)
	}
}

func TestCreateBranch(t *testing.T) {
	dir := t.TempDir()
	repoName := "test-repo"
	repoDir := filepath.Join(dir, repoName)
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	initLocalRepo(t, repoDir)

	r := NewRunner(dir)
	ctx := context.Background()

	branchName := "feature/test-123"
	if err := r.CreateBranch(ctx, repoName, branchName); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	got, err := r.CurrentBranch(ctx, repoName)
	if err != nil {
		t.Fatalf("CurrentBranch after create: %v", err)
	}
	if got != branchName {
		t.Errorf("expected branch %q, got %q", branchName, got)
	}
}

func TestCommitAndStatus(t *testing.T) {
	dir := t.TempDir()
	repoName := "test-repo"
	repoDir := filepath.Join(dir, repoName)
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	initLocalRepo(t, repoDir)

	r := NewRunner(dir)
	ctx := context.Background()

	// Create a new file
	f := filepath.Join(repoDir, "new.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Check status shows the new file
	files, err := r.Status(ctx, repoName)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected modified files, got none")
	}

	// Commit
	if err := r.Commit(ctx, repoName, "feat: add new file"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Status should be clean now
	files, err = r.Status(ctx, repoName)
	if err != nil {
		t.Fatalf("Status after commit: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected clean status, got %d files", len(files))
	}
}

func TestCreateBranch_AlreadyOnBranch(t *testing.T) {
	dir := t.TempDir()
	repoName := "test-repo"
	repoDir := filepath.Join(dir, repoName)
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	initLocalRepo(t, repoDir)

	r := NewRunner(dir)
	ctx := context.Background()

	branchName := "feature/test-123"
	// Create once
	if err := r.CreateBranch(ctx, repoName, branchName); err != nil {
		t.Fatalf("first CreateBranch: %v", err)
	}
	// Create again — should be a no-op
	if err := r.CreateBranch(ctx, repoName, branchName); err != nil {
		t.Fatalf("second CreateBranch: %v", err)
	}
}

func TestNewRunner_DefaultPath(t *testing.T) {
	r := NewRunner("")
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".getpod", "repos")
	if r.BasePath != want {
		t.Errorf("expected base path %q, got %q", want, r.BasePath)
	}
}

func TestCurrentBranch_NotARepo(t *testing.T) {
	r := NewRunner(t.TempDir())
	_, err := r.CurrentBranch(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent repo")
	}
}
