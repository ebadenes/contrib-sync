package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const defaultPageSize = 50

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	userAgent  string
}

type Repository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	CloneURL string `json:"clone_url"`
	SSHURL   string `json:"ssh_url"`
	Mirror   bool   `json:"mirror"`
	Fork     bool   `json:"fork"`
	Archived bool   `json:"archived"`
	Empty    bool   `json:"empty"`
	Private  bool   `json:"private"`
	Owner    Owner  `json:"owner"`
}

type Owner struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

type Version struct {
	Version string `json:"version"`
}

type NewClientOptions struct {
	BaseURL    string
	Token      string
	UserAgent  string
	HTTPClient *http.Client
}

func NewClient(opts NewClientOptions) *Client {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	userAgent := strings.TrimSpace(opts.UserAgent)
	if userAgent == "" {
		userAgent = "contrib-sync/dev"
	}

	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
		token: strings.TrimSpace(opts.Token),
		httpClient: httpClient,
		userAgent:  userAgent,
	}
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.Version(ctx)
	return err
}

func (c *Client) Version(ctx context.Context) (*Version, error) {
	var version Version
	if err := c.getJSON(ctx, "/api/v1/version", nil, &version); err != nil {
		return nil, err
	}
	return &version, nil
}

func (c *Client) ListUserRepos(ctx context.Context, username string) ([]Repository, error) {
	return c.listRepositories(ctx, "/api/v1/users/"+url.PathEscape(username)+"/repos")
}

func (c *Client) ListOrgRepos(ctx context.Context, org string) ([]Repository, error) {
	return c.listRepositories(ctx, "/api/v1/orgs/"+url.PathEscape(org)+"/repos")
}

func (c *Client) listRepositories(ctx context.Context, endpoint string) ([]Repository, error) {
	var repos []Repository
	page := 1

	for {
		var batch []Repository
		params := map[string]string{
			"page":  fmt.Sprintf("%d", page),
			"limit": fmt.Sprintf("%d", defaultPageSize),
		}
		if err := c.getJSON(ctx, endpoint, params, &batch); err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		repos = append(repos, batch...)
		if len(batch) < defaultPageSize {
			break
		}
		page++
	}

	return repos, nil
}

func (c *Client) FilterRepositories(repos []Repository, excludeForks, excludeArchived bool, excludeNames []string) []Repository {
	excluded := make(map[string]struct{}, len(excludeNames))
	for _, name := range excludeNames {
		excluded[strings.TrimSpace(name)] = struct{}{}
	}

	result := make([]Repository, 0, len(repos))
	for _, repo := range repos {
		if repo.Empty || repo.Mirror {
			continue
		}
		if excludeForks && repo.Fork {
			continue
		}
		if excludeArchived && repo.Archived {
			continue
		}
		if _, ok := excluded[repo.Name]; ok {
			continue
		}
		result = append(result, repo)
	}
	return result
}

func (c *Client) getJSON(ctx context.Context, endpoint string, params map[string]string, dst any) error {
	requestURL, err := c.buildURL(endpoint, params)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("create request %s: %w", requestURL, err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", requestURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("request %s failed with status %d: %s", requestURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode response from %s: %w", requestURL, err)
	}
	return nil
}

func (c *Client) buildURL(endpoint string, params map[string]string) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base url %q: %w", c.baseURL, err)
	}
	base.Path = path.Join(base.Path, endpoint)
	query := base.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	base.RawQuery = query.Encode()
	return base.String(), nil
}
