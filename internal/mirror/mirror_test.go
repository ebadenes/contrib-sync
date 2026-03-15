package mirror

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ebadenes/contrib-sync/internal/activity"
)

func TestEnsureCreatesGitRepository(t *testing.T) {
	repo := NewRepository(t.TempDir(), "alice@example.com")

	if err := repo.Ensure(context.Background()); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	output := runGitTest(t, repo.Dir, "rev-parse", "--is-inside-work-tree")
	if got, want := strings.TrimSpace(output), "true"; got != want {
		t.Fatalf("unexpected git repo state: got %q want %q", got, want)
	}
}

func TestWriteEventsCreatesEmptyCommitsAndReadsTimestamps(t *testing.T) {
	repo := NewRepository(t.TempDir(), "alice@example.com")
	t1 := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 15, 11, 30, 0, 0, time.UTC)

	created, err := repo.WriteEvents(context.Background(), []activity.Event{
		{Type: activity.TypeCommit, Repository: "alice/demo", Message: "primer evento", Timestamp: t1},
		{Type: activity.TypeIssue, Repository: "alice/demo", Message: "segundo evento", Timestamp: t2},
	})
	if err != nil {
		t.Fatalf("write events: %v", err)
	}
	if got, want := created, 2; got != want {
		t.Fatalf("unexpected created count: got %d want %d", got, want)
	}

	subjects := strings.Split(strings.TrimSpace(runGitTest(t, repo.Dir, "log", "--pretty=format:%s")), "\n")
	if got, want := len(subjects), 2; got != want {
		t.Fatalf("unexpected commit count in log: got %d want %d", got, want)
	}
	if got, want := subjects[0], "segundo evento"; got != want {
		t.Fatalf("unexpected latest subject: got %q want %q", got, want)
	}

	timestamps, err := repo.ExistingTimestamps(context.Background())
	if err != nil {
		t.Fatalf("existing timestamps: %v", err)
	}
	if got, want := len(timestamps), 2; got != want {
		t.Fatalf("unexpected timestamp count: got %d want %d", got, want)
	}
	if got, want := timestamps[0].UTC(), t2.UTC(); !got.Equal(want) {
		t.Fatalf("unexpected latest timestamp: got %s want %s", got, want)
	}
	if got, want := timestamps[1].UTC(), t1.UTC(); !got.Equal(want) {
		t.Fatalf("unexpected oldest timestamp: got %s want %s", got, want)
	}
}

func TestWriteEventsRequiresMirrorEmail(t *testing.T) {
	repo := NewRepository(t.TempDir(), "")
	_, err := repo.WriteEvents(context.Background(), []activity.Event{{Type: activity.TypeCommit, Timestamp: time.Now()}})
	if err == nil {
		t.Fatal("expected write events to fail without email")
	}
	if !strings.Contains(err.Error(), "mirror email is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteEventsSkipsTimestampsExcludedByActivityLayer(t *testing.T) {
	repo := NewRepository(t.TempDir(), "alice@example.com")
	t1 := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC)

	_, err := repo.WriteEvents(context.Background(), []activity.Event{
		{Type: activity.TypeCommit, Repository: "alice/demo", Message: "ya reflejado", Timestamp: t1},
	})
	if err != nil {
		t.Fatalf("initial write events: %v", err)
	}

	existing, err := repo.ExistingTimestamps(context.Background())
	if err != nil {
		t.Fatalf("existing timestamps: %v", err)
	}

	pending := activity.ExcludeTimestamps([]activity.Event{
		{Type: activity.TypeCommit, Repository: "alice/demo", Message: "duplicado", Timestamp: t1},
		{Type: activity.TypeIssue, Repository: "alice/demo", Message: "nuevo", Timestamp: t2},
	}, existing)

	created, err := repo.WriteEvents(context.Background(), pending)
	if err != nil {
		t.Fatalf("second write events: %v", err)
	}
	if got, want := created, 1; got != want {
		t.Fatalf("unexpected created count: got %d want %d", got, want)
	}

	logCount := strings.TrimSpace(runGitTest(t, repo.Dir, "rev-list", "--count", "HEAD"))
	if got, want := logCount, "2"; got != want {
		t.Fatalf("unexpected commit count: got %q want %q", got, want)
	}
}

func TestWriteFileCreatesMirrorReadme(t *testing.T) {
	repo := NewRepository(t.TempDir(), "alice@example.com")
	content := "# Contribution Mirror\n"

	if err := repo.WriteFile("README.md", content); err != nil {
		t.Fatalf("write file: %v", err)
	}

	data, err := os.ReadFile(repo.Dir + "/README.md")
	if err != nil {
		t.Fatalf("read readme: %v", err)
	}
	if got, want := string(data), content; got != want {
		t.Fatalf("unexpected readme content: got %q want %q", got, want)
	}
}

func TestExistingTimestampsReturnsEmptyWhenMirrorRepoDoesNotExist(t *testing.T) {
	repo := NewRepository(t.TempDir()+"/missing-mirror", "alice@example.com")

	timestamps, err := repo.ExistingTimestamps(context.Background())
	if err != nil {
		t.Fatalf("existing timestamps: %v", err)
	}
	if len(timestamps) != 0 {
		t.Fatalf("expected no timestamps, got %d", len(timestamps))
	}
}

func runGitTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output)
}
