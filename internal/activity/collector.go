package activity

import (
	"context"
	"fmt"
	"strings"

	"github.com/ebadenes/contrib-sync/internal/gitea"
)

type Source interface {
	ListCommits(ctx context.Context, owner, repo string, opts gitea.ListCommitOptions) ([]gitea.Commit, error)
	ListPullRequests(ctx context.Context, owner, repo string, opts gitea.ListPullRequestOptions) ([]gitea.PullRequest, error)
	ListIssues(ctx context.Context, owner, repo string, opts gitea.ListIssueOptions) ([]gitea.Issue, error)
	ListPullRequestReviews(ctx context.Context, owner, repo string, index int64) ([]gitea.Review, error)
}

type CollectOptions struct {
	Username      string
	Since         string
	ActivityTypes []string
	CopyMessages  bool
}

func Collect(ctx context.Context, source Source, repos []gitea.Repository, opts CollectOptions) ([]Event, error) {
	selected := selectedTypes(opts.ActivityTypes)
	collected := make([]Event, 0)

	for _, repo := range repos {
		if selected[TypeCommit] {
			events, err := collectCommits(ctx, source, repo, opts)
			if err != nil {
				return nil, err
			}
			collected = append(collected, events...)
		}

		if selected[TypePR] {
			events, err := collectPullRequests(ctx, source, repo, opts)
			if err != nil {
				return nil, err
			}
			collected = append(collected, events...)
		}

		if selected[TypeIssue] {
			events, err := collectIssues(ctx, source, repo, opts)
			if err != nil {
				return nil, err
			}
			collected = append(collected, events...)
		}

		if selected[TypeReview] {
			events, err := collectReviews(ctx, source, repo, opts)
			if err != nil {
				return nil, err
			}
			collected = append(collected, events...)
		}
	}

	return Deduplicate(collected), nil
}

func collectCommits(ctx context.Context, source Source, repo gitea.Repository, opts CollectOptions) ([]Event, error) {
	commits, err := source.ListCommits(ctx, repo.Owner.Login, repo.Name, gitea.ListCommitOptions{
		Since:  opts.Since,
		Author: strings.TrimSpace(opts.Username),
	})
	if err != nil {
		return nil, fmt.Errorf("collect commits for %s: %w", repo.FullName, err)
	}

	events := make([]Event, 0, len(commits))
	for _, commit := range commits {
		if event, ok := NewCommitEvent(repo, commit, opts.CopyMessages); ok {
			events = append(events, event)
		}
	}
	return events, nil
}

func collectPullRequests(ctx context.Context, source Source, repo gitea.Repository, opts CollectOptions) ([]Event, error) {
	pulls, err := source.ListPullRequests(ctx, repo.Owner.Login, repo.Name, gitea.ListPullRequestOptions{
		State:  "all",
		Poster: strings.TrimSpace(opts.Username),
	})
	if err != nil {
		return nil, fmt.Errorf("collect pull requests for %s: %w", repo.FullName, err)
	}

	events := make([]Event, 0, len(pulls))
	for _, pr := range pulls {
		if event, ok := NewPullRequestEvent(repo, pr, opts.CopyMessages); ok {
			events = append(events, event)
		}
	}
	return events, nil
}

func collectIssues(ctx context.Context, source Source, repo gitea.Repository, opts CollectOptions) ([]Event, error) {
	issues, err := source.ListIssues(ctx, repo.Owner.Login, repo.Name, gitea.ListIssueOptions{
		State:     "all",
		CreatedBy: strings.TrimSpace(opts.Username),
		Since:     opts.Since,
		Type:      "issues",
	})
	if err != nil {
		return nil, fmt.Errorf("collect issues for %s: %w", repo.FullName, err)
	}

	events := make([]Event, 0, len(issues))
	for _, issue := range issues {
		if event, ok := NewIssueEvent(repo, issue, opts.CopyMessages); ok {
			events = append(events, event)
		}
	}
	return events, nil
}

func collectReviews(ctx context.Context, source Source, repo gitea.Repository, opts CollectOptions) ([]Event, error) {
	pulls, err := source.ListPullRequests(ctx, repo.Owner.Login, repo.Name, gitea.ListPullRequestOptions{State: "all"})
	if err != nil {
		return nil, fmt.Errorf("collect review pull requests for %s: %w", repo.FullName, err)
	}

	events := make([]Event, 0)
	username := strings.TrimSpace(opts.Username)
	for _, pr := range pulls {
		prNumber := pr.NumberOrIndex()
		if prNumber <= 0 {
			continue
		}
		reviews, err := source.ListPullRequestReviews(ctx, repo.Owner.Login, repo.Name, prNumber)
		if err != nil {
			return nil, fmt.Errorf("collect reviews for %s pull request #%d: %w", repo.FullName, prNumber, err)
		}
		for _, review := range reviews {
			if review.User == nil || strings.TrimSpace(review.User.Login) != username {
				continue
			}
			if event, ok := NewReviewEvent(repo, pr, review, opts.CopyMessages); ok {
				events = append(events, event)
			}
		}
	}

	return events, nil
}

func selectedTypes(types []string) map[string]bool {
	selected := map[string]bool{}
	for _, activityType := range types {
		selected[strings.TrimSpace(activityType)] = true
	}
	return selected
}
