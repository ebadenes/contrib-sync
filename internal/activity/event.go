package activity

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ebadenes/contrib-sync/internal/gitea"
)

const (
	TypeCommit = "commits"
	TypePR     = "prs"
	TypeIssue  = "issues"
	TypeReview = "reviews"
)

type Event struct {
	Type       string
	Repository string
	Owner      string
	Name       string
	SourceID   string
	Title      string
	Message    string
	Timestamp  time.Time
}

func (e Event) Key() string {
	return strings.Join([]string{e.Type, e.Repository, e.SourceID, normalizeTimestamp(e.Timestamp)}, "|")
}

func Sort(events []Event) {
	sort.Slice(events, func(i, j int) bool {
		left := events[i].Timestamp.UTC()
		right := events[j].Timestamp.UTC()
		if !left.Equal(right) {
			return left.Before(right)
		}
		if events[i].Type != events[j].Type {
			return events[i].Type < events[j].Type
		}
		if events[i].Repository != events[j].Repository {
			return events[i].Repository < events[j].Repository
		}
		return events[i].SourceID < events[j].SourceID
	})
}

func CountByType(events []Event) map[string]int {
	counts := map[string]int{
		TypeCommit: 0,
		TypePR:     0,
		TypeIssue:  0,
		TypeReview: 0,
	}
	for _, event := range events {
		counts[event.Type]++
	}
	return counts
}

func NewCommitEvent(repo gitea.Repository, commit gitea.Commit, copyMessages bool) (Event, bool) {
	timestamp := commit.Commit.Author.Date
	if timestamp.IsZero() {
		timestamp = commit.Commit.Committer.Date
	}
	if timestamp.IsZero() {
		return Event{}, false
	}

	title := firstLine(commit.Commit.Message)
	message := defaultMessage(TypeCommit, repo.FullName, commit.SHA)
	if copyMessages && strings.TrimSpace(commit.Commit.Message) != "" {
		message = strings.TrimSpace(commit.Commit.Message)
	}

	return Event{
		Type:       TypeCommit,
		Repository: repo.FullName,
		Owner:      repo.Owner.Login,
		Name:       repo.Name,
		SourceID:   commit.SHA,
		Title:      title,
		Message:    message,
		Timestamp:  timestamp.UTC(),
	}, true
}

func NewPullRequestEvent(repo gitea.Repository, pr gitea.PullRequest, copyMessages bool) (Event, bool) {
	prNumber := pr.NumberOrIndex()
	if pr.CreatedAt.IsZero() || prNumber <= 0 {
		return Event{}, false
	}

	message := defaultMessage(TypePR, repo.FullName, strconv.FormatInt(prNumber, 10))
	if copyMessages && strings.TrimSpace(pr.Title) != "" {
		message = strings.TrimSpace(pr.Title)
	}

	return Event{
		Type:       TypePR,
		Repository: repo.FullName,
		Owner:      repo.Owner.Login,
		Name:       repo.Name,
		SourceID:   strconv.FormatInt(prNumber, 10),
		Title:      strings.TrimSpace(pr.Title),
		Message:    message,
		Timestamp:  pr.CreatedAt.UTC(),
	}, true
}

func NewIssueEvent(repo gitea.Repository, issue gitea.Issue, copyMessages bool) (Event, bool) {
	if issue.CreatedAt.IsZero() {
		return Event{}, false
	}

	message := defaultMessage(TypeIssue, repo.FullName, strconv.FormatInt(issue.Index, 10))
	if copyMessages && strings.TrimSpace(issue.Title) != "" {
		message = strings.TrimSpace(issue.Title)
	}

	return Event{
		Type:       TypeIssue,
		Repository: repo.FullName,
		Owner:      repo.Owner.Login,
		Name:       repo.Name,
		SourceID:   strconv.FormatInt(issue.Index, 10),
		Title:      strings.TrimSpace(issue.Title),
		Message:    message,
		Timestamp:  issue.CreatedAt.UTC(),
	}, true
}

func NewReviewEvent(repo gitea.Repository, pr gitea.PullRequest, review gitea.Review, copyMessages bool) (Event, bool) {
	prNumber := pr.NumberOrIndex()
	if review.SubmittedAt == nil || review.SubmittedAt.IsZero() || prNumber <= 0 {
		return Event{}, false
	}

	sourceID := fmt.Sprintf("%d:%d", prNumber, review.ID)
	message := defaultMessage(TypeReview, repo.FullName, sourceID)
	if copyMessages && strings.TrimSpace(review.Body) != "" {
		message = strings.TrimSpace(review.Body)
	}

	return Event{
		Type:       TypeReview,
		Repository: repo.FullName,
		Owner:      repo.Owner.Login,
		Name:       repo.Name,
		SourceID:   sourceID,
		Title:      strings.TrimSpace(review.State),
		Message:    message,
		Timestamp:  review.SubmittedAt.UTC(),
	}, true
}

func defaultMessage(activityType, repository, sourceID string) string {
	return fmt.Sprintf("mirror %s activity from %s (%s)", activityType, repository, sourceID)
}

func firstLine(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "\n")
	return strings.TrimSpace(parts[0])
}

func normalizeTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
