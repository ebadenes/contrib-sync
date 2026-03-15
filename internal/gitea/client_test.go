package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestVersionUsesAuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "token secret"; got != want {
			t.Fatalf("unexpected auth header: got %q want %q", got, want)
		}
		_ = json.NewEncoder(w).Encode(Version{Version: "1.24.0"})
	}))
	defer server.Close()

	client := NewClient(NewClientOptions{BaseURL: server.URL, Token: "secret"})
	version, err := client.Version(context.Background())
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if got, want := version.Version, "1.24.0"; got != want {
		t.Fatalf("unexpected version: got %q want %q", got, want)
	}
}

func TestListUserReposPaginates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/users/alice/repos") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit != defaultPageSize {
			t.Fatalf("unexpected limit: %d", limit)
		}

		batch := make([]Repository, 0)
		switch page {
		case 1:
			for i := 0; i < defaultPageSize; i++ {
				batch = append(batch, Repository{Name: "repo-" + strconv.Itoa(i+1)})
			}
		case 2:
			batch = append(batch, Repository{Name: "repo-last"})
		}
		_ = json.NewEncoder(w).Encode(batch)
	}))
	defer server.Close()

	client := NewClient(NewClientOptions{BaseURL: server.URL, Token: "secret"})
	repos, err := client.ListUserRepos(context.Background(), "alice")
	if err != nil {
		t.Fatalf("list user repos: %v", err)
	}
	if got, want := len(repos), defaultPageSize+1; got != want {
		t.Fatalf("unexpected repo count: got %d want %d", got, want)
	}
}

func TestFilterRepositories(t *testing.T) {
	client := NewClient(NewClientOptions{BaseURL: "https://gitea.example.com"})
	repos := []Repository{
		{Name: "keep", Owner: Owner{Login: "alice"}},
		{Name: "forked", Fork: true},
		{Name: "archived", Archived: true},
		{Name: "empty", Empty: true},
		{Name: "mirror", Mirror: true},
		{Name: "blocked"},
	}

	filtered := client.FilterRepositories(repos, true, true, []string{"blocked"})
	if got, want := len(filtered), 1; got != want {
		t.Fatalf("unexpected filtered count: got %d want %d", got, want)
	}
	if got, want := filtered[0].Name, "keep"; got != want {
		t.Fatalf("unexpected repo kept: got %q want %q", got, want)
	}
}
