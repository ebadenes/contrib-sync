package activity

import (
	"testing"
	"time"

	"github.com/ebadenes/contrib-sync/internal/gitea"
)

func TestNewCommitEventUsesCommitMessageWhenEnabled(t *testing.T) {
	repo := gitea.Repository{Name: "demo", FullName: "alice/demo", Owner: gitea.Owner{Login: "alice"}}
	timestamp := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	event, ok := NewCommitEvent(repo, gitea.Commit{
		SHA: "abc123",
		Commit: gitea.CommitInfo{
			Author:  gitea.CommitSignature{Date: timestamp},
			Message: "Add sync collector\n\nMore details",
		},
	}, true)
	if !ok {
		t.Fatal("expected commit event to be created")
	}
	if got, want := event.Type, TypeCommit; got != want {
		t.Fatalf("unexpected type: got %q want %q", got, want)
	}
	if got, want := event.Title, "Add sync collector"; got != want {
		t.Fatalf("unexpected title: got %q want %q", got, want)
	}
	if got, want := event.Message, "Add sync collector\n\nMore details"; got != want {
		t.Fatalf("unexpected message: got %q want %q", got, want)
	}
}

func TestNewReviewEventUsesFallbackMessageWhenBodyMissing(t *testing.T) {
	repo := gitea.Repository{Name: "demo", FullName: "alice/demo", Owner: gitea.Owner{Login: "alice"}}
	submittedAt := time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC)

	event, ok := NewReviewEvent(repo, gitea.PullRequest{Number: 42}, gitea.Review{
		ID:          99,
		State:       "APPROVED",
		SubmittedAt: &submittedAt,
	}, true)
	if !ok {
		t.Fatal("expected review event to be created")
	}
	if got, want := event.SourceID, "42:99"; got != want {
		t.Fatalf("unexpected source id: got %q want %q", got, want)
	}
	if got, want := event.Message, "mirror reviews activity from alice/demo (42:99)"; got != want {
		t.Fatalf("unexpected message: got %q want %q", got, want)
	}
}

func TestDeduplicateRemovesRepeatedEventsAndSorts(t *testing.T) {
	later := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	earlier := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)

	events := []Event{
		{Type: TypeCommit, Repository: "alice/demo", SourceID: "b", Timestamp: later},
		{Type: TypeCommit, Repository: "alice/demo", SourceID: "a", Timestamp: earlier},
		{Type: TypeCommit, Repository: "alice/demo", SourceID: "a", Timestamp: earlier},
	}

	result := Deduplicate(events)
	if got, want := len(result), 2; got != want {
		t.Fatalf("unexpected event count: got %d want %d", got, want)
	}
	if got, want := result[0].SourceID, "a"; got != want {
		t.Fatalf("unexpected first event: got %q want %q", got, want)
	}
	if got, want := result[1].SourceID, "b"; got != want {
		t.Fatalf("unexpected second event: got %q want %q", got, want)
	}
}

func TestExcludeTimestampsRemovesMirroredEvents(t *testing.T) {
	t1 := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)

	events := []Event{
		{Type: TypeCommit, Repository: "alice/demo", SourceID: "a", Timestamp: t1},
		{Type: TypePR, Repository: "alice/demo", SourceID: "2", Timestamp: t2},
	}

	result := ExcludeTimestamps(events, []time.Time{t1})
	if got, want := len(result), 1; got != want {
		t.Fatalf("unexpected event count: got %d want %d", got, want)
	}
	if got, want := result[0].SourceID, "2"; got != want {
		t.Fatalf("unexpected remaining event: got %q want %q", got, want)
	}
}
