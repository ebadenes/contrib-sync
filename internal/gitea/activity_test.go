package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestListCommitsUsesFiltersAndPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/api/v1/repos/alice/demo/commits"; got != want {
			t.Fatalf("unexpected path: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("since"), "2024-01-01T00:00:00Z"; got != want {
			t.Fatalf("unexpected since: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("until"), "2024-01-31T23:59:59Z"; got != want {
			t.Fatalf("unexpected until: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("author"), "alice@example.com"; got != want {
			t.Fatalf("unexpected author: got %q want %q", got, want)
		}
		for key, want := range map[string]string{"stat": "false", "verification": "false", "files": "false"} {
			if got := r.URL.Query().Get(key); got != want {
				t.Fatalf("unexpected %s: got %q want %q", key, got, want)
			}
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit != defaultPageSize {
			t.Fatalf("unexpected limit: got %d want %d", limit, defaultPageSize)
		}

		batch := make([]Commit, 0)
		switch page {
		case 1:
			for i := 0; i < defaultPageSize; i++ {
				batch = append(batch, Commit{SHA: "sha-" + strconv.Itoa(i+1)})
			}
		case 2:
			batch = append(batch, Commit{SHA: "sha-last"})
		}
		_ = json.NewEncoder(w).Encode(batch)
	}))
	defer server.Close()

	client := NewClient(NewClientOptions{BaseURL: server.URL, Token: "secret"})
	commits, err := client.ListCommits(context.Background(), "alice", "demo", ListCommitOptions{
		Since:  "2024-01-01T00:00:00Z",
		Until:  "2024-01-31T23:59:59Z",
		Author: "alice@example.com",
	})
	if err != nil {
		t.Fatalf("list commits: %v", err)
	}
	if got, want := len(commits), defaultPageSize+1; got != want {
		t.Fatalf("unexpected commit count: got %d want %d", got, want)
	}
	if got, want := commits[len(commits)-1].SHA, "sha-last"; got != want {
		t.Fatalf("unexpected last sha: got %q want %q", got, want)
	}
}

func TestListPullRequestsUsesPosterAndState(t *testing.T) {
	timestamp := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/api/v1/repos/alice/demo/pulls"; got != want {
			t.Fatalf("unexpected path: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("state"), "all"; got != want {
			t.Fatalf("unexpected state: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("poster"), "alice"; got != want {
			t.Fatalf("unexpected poster: got %q want %q", got, want)
		}

		_ = json.NewEncoder(w).Encode([]PullRequest{{
			ID:        10,
			Number:    7,
			Title:     "Add sync status",
			State:     "closed",
			Merged:    true,
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
		}})
	}))
	defer server.Close()

	client := NewClient(NewClientOptions{BaseURL: server.URL})
	pulls, err := client.ListPullRequests(context.Background(), "alice", "demo", ListPullRequestOptions{Poster: "alice"})
	if err != nil {
		t.Fatalf("list pull requests: %v", err)
	}
	if got, want := len(pulls), 1; got != want {
		t.Fatalf("unexpected pull request count: got %d want %d", got, want)
	}
	if got, want := pulls[0].NumberOrIndex(), int64(7); got != want {
		t.Fatalf("unexpected pull request number: got %d want %d", got, want)
	}
	if pulls[0].CreatedAt.IsZero() {
		t.Fatal("expected created_at to be parsed")
	}
}

func TestListIssuesUsesCreatedBySinceAndType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/api/v1/repos/alice/demo/issues"; got != want {
			t.Fatalf("unexpected path: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("created_by"), "alice"; got != want {
			t.Fatalf("unexpected created_by: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("since"), "2024-01-01T00:00:00Z"; got != want {
			t.Fatalf("unexpected since: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("type"), "issues"; got != want {
			t.Fatalf("unexpected type: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("state"), "all"; got != want {
			t.Fatalf("unexpected state: got %q want %q", got, want)
		}

		_ = json.NewEncoder(w).Encode([]Issue{{ID: 20, Index: 3, Title: "Bug report", State: "open"}})
	}))
	defer server.Close()

	client := NewClient(NewClientOptions{BaseURL: server.URL})
	issues, err := client.ListIssues(context.Background(), "alice", "demo", ListIssueOptions{
		CreatedBy: "alice",
		Since:     "2024-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}
	if got, want := len(issues), 1; got != want {
		t.Fatalf("unexpected issue count: got %d want %d", got, want)
	}
	if got, want := issues[0].Index, int64(3); got != want {
		t.Fatalf("unexpected issue index: got %d want %d", got, want)
	}
}

func TestListPullRequestReviewsUsesReviewEndpoint(t *testing.T) {
	submittedAt := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/api/v1/repos/alice/demo/pulls/42/reviews"; got != want {
			t.Fatalf("unexpected path: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("page"), "1"; got != want {
			t.Fatalf("unexpected page: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("limit"), strconv.Itoa(defaultPageSize); got != want {
			t.Fatalf("unexpected limit: got %q want %q", got, want)
		}

		_ = json.NewEncoder(w).Encode([]Review{{
			ID:          99,
			State:       "APPROVED",
			CommitID:    "abc123",
			SubmittedAt: &submittedAt,
		}})
	}))
	defer server.Close()

	client := NewClient(NewClientOptions{BaseURL: server.URL})
	reviews, err := client.ListPullRequestReviews(context.Background(), "alice", "demo", 42)
	if err != nil {
		t.Fatalf("list reviews: %v", err)
	}
	if got, want := len(reviews), 1; got != want {
		t.Fatalf("unexpected review count: got %d want %d", got, want)
	}
	if reviews[0].SubmittedAt == nil || reviews[0].SubmittedAt.IsZero() {
		t.Fatal("expected submitted_at to be parsed")
	}
}
