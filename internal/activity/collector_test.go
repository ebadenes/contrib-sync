package activity

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ebadenes/contrib-sync/internal/gitea"
)

type fakeSource struct {
	commits      map[string][]gitea.Commit
	pulls        map[string][]gitea.PullRequest
	issues       map[string][]gitea.Issue
	reviews      map[string][]gitea.Review
	commitCalls  []gitea.ListCommitOptions
	pullCalls    []gitea.ListPullRequestOptions
	issueCalls   []gitea.ListIssueOptions
	reviewCalls  []int64
}

func (f *fakeSource) ListCommits(_ context.Context, owner, repo string, opts gitea.ListCommitOptions) ([]gitea.Commit, error) {
	f.commitCalls = append(f.commitCalls, opts)
	return f.commits[key(owner, repo)], nil
}

func (f *fakeSource) ListPullRequests(_ context.Context, owner, repo string, opts gitea.ListPullRequestOptions) ([]gitea.PullRequest, error) {
	f.pullCalls = append(f.pullCalls, opts)
	return f.pulls[key(owner, repo)], nil
}

func (f *fakeSource) ListIssues(_ context.Context, owner, repo string, opts gitea.ListIssueOptions) ([]gitea.Issue, error) {
	f.issueCalls = append(f.issueCalls, opts)
	return f.issues[key(owner, repo)], nil
}

func (f *fakeSource) ListPullRequestReviews(_ context.Context, owner, repo string, index int64) ([]gitea.Review, error) {
	f.reviewCalls = append(f.reviewCalls, index)
	return f.reviews[fmt.Sprintf("%s#%d", key(owner, repo), index)], nil
}

func TestCollectHonorsSelectedTypesAndBuildsEvents(t *testing.T) {
	timestamp := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	repo := gitea.Repository{Name: "demo", FullName: "alice/demo", Owner: gitea.Owner{Login: "alice"}}
	source := &fakeSource{
		commits: map[string][]gitea.Commit{
			"alice/demo": {{SHA: "abc123", Commit: gitea.CommitInfo{Author: gitea.CommitSignature{Date: timestamp}, Message: "Add collector"}}},
		},
		issues: map[string][]gitea.Issue{
			"alice/demo": {{Index: 5, Title: "Bug", CreatedAt: timestamp.Add(time.Hour)}},
		},
	}

	events, err := Collect(context.Background(), source, []gitea.Repository{repo}, CollectOptions{
		Username:      "alice",
		Since:         "2024-01-01T00:00:00Z",
		ActivityTypes: []string{TypeCommit, TypeIssue},
		CopyMessages:  true,
	})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if got, want := len(events), 2; got != want {
		t.Fatalf("unexpected event count: got %d want %d", got, want)
	}
	if got, want := len(source.pullCalls), 0; got != want {
		t.Fatalf("unexpected pull calls: got %d want %d", got, want)
	}
	if got, want := len(source.reviewCalls), 0; got != want {
		t.Fatalf("unexpected review calls: got %d want %d", got, want)
	}
	if got, want := source.commitCalls[0].Author, "alice"; got != want {
		t.Fatalf("unexpected commit author filter: got %q want %q", got, want)
	}
	if got, want := source.issueCalls[0].CreatedBy, "alice"; got != want {
		t.Fatalf("unexpected issue created_by filter: got %q want %q", got, want)
	}
}

func TestCollectFiltersReviewsByUsername(t *testing.T) {
	timestamp := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	repo := gitea.Repository{Name: "demo", FullName: "alice/demo", Owner: gitea.Owner{Login: "alice"}}
	source := &fakeSource{
		pulls: map[string][]gitea.PullRequest{
			"alice/demo": {{Number: 9, Title: "PR", CreatedAt: timestamp}},
		},
		reviews: map[string][]gitea.Review{
			"alice/demo#9": {
				{ID: 1, State: "APPROVED", SubmittedAt: refTime(timestamp), User: &gitea.Owner{Login: "alice"}},
				{ID: 2, State: "COMMENTED", SubmittedAt: refTime(timestamp.Add(time.Minute)), User: &gitea.Owner{Login: "bob"}},
			},
		},
	}

	events, err := Collect(context.Background(), source, []gitea.Repository{repo}, CollectOptions{
		Username:      "alice",
		ActivityTypes: []string{TypeReview},
	})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("unexpected event count: got %d want %d", got, want)
	}
	if got, want := events[0].SourceID, "9:1"; got != want {
		t.Fatalf("unexpected review source id: got %q want %q", got, want)
	}
	if got, want := len(source.reviewCalls), 1; got != want {
		t.Fatalf("unexpected review calls: got %d want %d", got, want)
	}
}

func key(owner, repo string) string {
	return owner + "/" + repo
}

func refTime(value time.Time) *time.Time {
	return &value
}
