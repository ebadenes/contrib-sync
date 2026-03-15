package gitea

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Commit struct {
	SHA       string      `json:"sha"`
	Commit    CommitInfo  `json:"commit"`
	Author    *Owner      `json:"author"`
	Committer *Owner      `json:"committer"`
	Parents   []CommitRef `json:"parents"`
}

type CommitInfo struct {
	Author    CommitSignature `json:"author"`
	Committer CommitSignature `json:"committer"`
	Message   string          `json:"message"`
	Tree      CommitRef       `json:"tree"`
}

type CommitSignature struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

type CommitRef struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

type ListCommitOptions struct {
	Since  string
	Until  string
	Author string
}

type PullRequest struct {
	ID        int64      `json:"id"`
	Index     int64      `json:"index"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	Draft     bool       `json:"draft"`
	Merged    bool       `json:"merged"`
	User      *Owner     `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
	MergedAt  *time.Time `json:"merged_at"`
}

type ListPullRequestOptions struct {
	State  string
	Poster string
}

type Issue struct {
	ID        int64      `json:"id"`
	Index     int64      `json:"index"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	User      *Owner     `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
}

type ListIssueOptions struct {
	State     string
	CreatedBy string
	Since     string
	Type      string
}

type Review struct {
	ID          int64      `json:"id"`
	Body        string     `json:"body"`
	State       string     `json:"state"`
	CommitID    string     `json:"commit_id"`
	User        *Owner     `json:"user"`
	SubmittedAt *time.Time `json:"submitted_at"`
}

func (c *Client) ListCommits(ctx context.Context, owner, repo string, opts ListCommitOptions) ([]Commit, error) {
	params := map[string]string{
		"stat":         "false",
		"verification": "false",
		"files":        "false",
	}
	if value := strings.TrimSpace(opts.Since); value != "" {
		params["since"] = value
	}
	if value := strings.TrimSpace(opts.Until); value != "" {
		params["until"] = value
	}
	if value := strings.TrimSpace(opts.Author); value != "" {
		params["author"] = value
	}

	return listPaginated[Commit](ctx, c, repositoryEndpoint(owner, repo, "/commits"), params)
}

func (c *Client) ListPullRequests(ctx context.Context, owner, repo string, opts ListPullRequestOptions) ([]PullRequest, error) {
	params := map[string]string{
		"state": "all",
	}
	if value := strings.TrimSpace(opts.State); value != "" {
		params["state"] = value
	}
	if value := strings.TrimSpace(opts.Poster); value != "" {
		params["poster"] = value
	}

	return listPaginated[PullRequest](ctx, c, repositoryEndpoint(owner, repo, "/pulls"), params)
}

func (c *Client) ListIssues(ctx context.Context, owner, repo string, opts ListIssueOptions) ([]Issue, error) {
	params := map[string]string{
		"state": "all",
		"type":  "issues",
	}
	if value := strings.TrimSpace(opts.State); value != "" {
		params["state"] = value
	}
	if value := strings.TrimSpace(opts.CreatedBy); value != "" {
		params["created_by"] = value
	}
	if value := strings.TrimSpace(opts.Since); value != "" {
		params["since"] = value
	}
	if value := strings.TrimSpace(opts.Type); value != "" {
		params["type"] = value
	}

	return listPaginated[Issue](ctx, c, repositoryEndpoint(owner, repo, "/issues"), params)
}

func (c *Client) ListPullRequestReviews(ctx context.Context, owner, repo string, index int64) ([]Review, error) {
	return listPaginated[Review](ctx, c, repositoryEndpoint(owner, repo, fmt.Sprintf("/pulls/%d/reviews", index)), nil)
}

func listPaginated[T any](ctx context.Context, client *Client, endpoint string, params map[string]string) ([]T, error) {
	items := make([]T, 0)
	page := 1

	for {
		batchParams := copyParams(params)
		batchParams["page"] = strconv.Itoa(page)
		batchParams["limit"] = strconv.Itoa(defaultPageSize)

		var batch []T
		if err := client.getJSON(ctx, endpoint, batchParams, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		items = append(items, batch...)
		if len(batch) < defaultPageSize {
			break
		}
		page++
	}

	return items, nil
}

func repositoryEndpoint(owner, repo, suffix string) string {
	return "/api/v1/repos/" + url.PathEscape(strings.TrimSpace(owner)) + "/" + url.PathEscape(strings.TrimSpace(repo)) + suffix
}

func copyParams(params map[string]string) map[string]string {
	if len(params) == 0 {
		return make(map[string]string)
	}

	cloned := make(map[string]string, len(params)+2)
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}
