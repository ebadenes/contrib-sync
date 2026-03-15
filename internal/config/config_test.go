package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesDefaultsAndNormalizes(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	content := `gitea:
  url: "https://gitea.example.com/"
  token: "token-value"
  username: "alice"

mirror:
  dir: "~/mirror"
  email: "alice@example.com"

sync:
  since: "2024-01-01T00:00:00Z"
`

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got, want := cfg.Gitea.URL, "https://gitea.example.com"; got != want {
		t.Fatalf("unexpected gitea url: got %q want %q", got, want)
	}
	if got, want := cfg.Mirror.Remote, "origin"; got != want {
		t.Fatalf("unexpected mirror remote: got %q want %q", got, want)
	}
	if len(cfg.Sync.ActivityTypes) != 4 {
		t.Fatalf("expected default activity types, got %v", cfg.Sync.ActivityTypes)
	}
	if !strings.Contains(cfg.Mirror.Dir, "mirror") {
		t.Fatalf("expected expanded mirror dir, got %q", cfg.Mirror.Dir)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("CONTRIB_SYNC_GITEA_TOKEN", "override-token")
	t.Setenv("CONTRIB_SYNC_GITEA_ORGS", "org-a, org-b")
	t.Setenv("CONTRIB_SYNC_SYNC_ACTIVITY_TYPES", "reviews,commits")
	t.Setenv("CONTRIB_SYNC_SYNC_COPY_MESSAGES", "true")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	content := `gitea:
  url: "https://gitea.example.com"
  token: "token-value"
  username: "alice"

mirror:
  dir: "/mirror"
  email: "alice@example.com"
  remote: "origin"

sync:
  since: "2024-01-01T00:00:00Z"
  activity_types: [commits]
  copy_messages: false
`

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got, want := cfg.Gitea.Token, "override-token"; got != want {
		t.Fatalf("unexpected token: got %q want %q", got, want)
	}
	if len(cfg.Gitea.Orgs) != 2 {
		t.Fatalf("expected 2 orgs, got %v", cfg.Gitea.Orgs)
	}
	if !cfg.Sync.CopyMessages {
		t.Fatalf("expected copy_messages to be true")
	}
	if got := strings.Join(cfg.Sync.ActivityTypes, ","); got != "commits,reviews" {
		t.Fatalf("unexpected activity types: %q", got)
	}
}

func TestLoadRejectsInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	content := `gitea:
  url: ""
  token: ""
  username: ""

mirror:
  dir: ""
  email: ""
  remote: ""

sync:
  since: "invalid"
  activity_types: [unknown]
`

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
