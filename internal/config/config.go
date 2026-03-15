package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultConfigPath = "config.yaml"
	envPrefix         = "CONTRIB_SYNC_"
)

var validActivityTypes = map[string]struct{}{
	"commits": {},
	"prs":     {},
	"issues":  {},
	"reviews": {},
}

type Config struct {
	Gitea  GiteaConfig  `yaml:"gitea"`
	Mirror MirrorConfig `yaml:"mirror"`
	Sync   SyncConfig   `yaml:"sync"`
}

type GiteaConfig struct {
	URL      string   `yaml:"url"`
	Token    string   `yaml:"token"`
	Username string   `yaml:"username"`
	Orgs     []string `yaml:"orgs"`
}

type MirrorConfig struct {
	Dir    string `yaml:"dir"`
	Email  string `yaml:"email"`
	Remote string `yaml:"remote"`
}

type SyncConfig struct {
	Since           string   `yaml:"since"`
	ActivityTypes   []string `yaml:"activity_types"`
	CopyMessages    bool     `yaml:"copy_messages"`
	ExcludeRepos    []string `yaml:"exclude_repos"`
	ExcludeForks    bool     `yaml:"exclude_forks"`
	ExcludeArchived bool     `yaml:"exclude_archived"`
}

func DefaultConfigPath() string {
	return defaultConfigPath
}

func ExampleConfig() string {
	return strings.TrimSpace(`gitea:
  url: "https://gitea.example.com"
  token: "replace-with-token"
  username: "your-username"
  orgs: []

mirror:
  dir: "/mirror"
  email: "your-github-email@example.com"
  remote: "origin"

sync:
  since: "2024-01-01T00:00:00Z"
  activity_types:
    - commits
    - prs
    - issues
    - reviews
  copy_messages: false
  exclude_repos: []
  exclude_forks: true
  exclude_archived: true
`) + "\n"
}

func Load(path string) (*Config, error) {
	resolvedPath := strings.TrimSpace(path)
	if resolvedPath == "" {
		resolvedPath = DefaultConfigPath()
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", resolvedPath, err)
	}

	cfg := &Config{}
	if err := parseConfig(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", resolvedPath, err)
	}

	cfg.applyDefaults()
	cfg.applyEnvOverrides()
	cfg.normalize()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}

	if strings.TrimSpace(c.Gitea.URL) == "" {
		return errors.New("gitea.url is required")
	}
	if strings.TrimSpace(c.Gitea.Token) == "" {
		return errors.New("gitea.token is required")
	}
	if strings.TrimSpace(c.Gitea.Username) == "" {
		return errors.New("gitea.username is required")
	}
	if strings.TrimSpace(c.Mirror.Dir) == "" {
		return errors.New("mirror.dir is required")
	}
	if strings.TrimSpace(c.Mirror.Email) == "" {
		return errors.New("mirror.email is required")
	}
	if strings.TrimSpace(c.Mirror.Remote) == "" {
		return errors.New("mirror.remote is required")
	}
	if strings.TrimSpace(c.Sync.Since) == "" {
		return errors.New("sync.since is required")
	}
	if _, err := time.Parse(time.RFC3339, c.Sync.Since); err != nil {
		return fmt.Errorf("sync.since must be RFC3339: %w", err)
	}
	if len(c.Sync.ActivityTypes) == 0 {
		return errors.New("sync.activity_types must contain at least one value")
	}

	for _, value := range c.Sync.ActivityTypes {
		if _, ok := validActivityTypes[value]; !ok {
			return fmt.Errorf("sync.activity_types contains unsupported value %q", value)
		}
	}

	return nil
}

func (c *Config) ConfigSummary() string {
	return fmt.Sprintf(
		"Gitea URL: %s\nGitea Username: %s\nGitea Orgs: %s\nMirror Dir: %s\nMirror Email: %s\nMirror Remote: %s\nSince: %s\nActivity Types: %s\nExclude Forks: %t\nExclude Archived: %t\nCopy Messages: %t",
		c.Gitea.URL,
		c.Gitea.Username,
		joinOrNone(c.Gitea.Orgs),
		c.Mirror.Dir,
		c.Mirror.Email,
		c.Mirror.Remote,
		c.Sync.Since,
		strings.Join(c.Sync.ActivityTypes, ", "),
		c.Sync.ExcludeForks,
		c.Sync.ExcludeArchived,
		c.Sync.CopyMessages,
	)
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.Mirror.Remote) == "" {
		c.Mirror.Remote = "origin"
	}
	if len(c.Sync.ActivityTypes) == 0 {
		c.Sync.ActivityTypes = []string{"commits", "prs", "issues", "reviews"}
	}
	if strings.TrimSpace(c.Sync.Since) == "" {
		c.Sync.Since = "2024-01-01T00:00:00Z"
	}
}

func (c *Config) applyEnvOverrides() {
	applyStringEnv(&c.Gitea.URL, envPrefix+"GITEA_URL")
	applyStringEnv(&c.Gitea.Token, envPrefix+"GITEA_TOKEN")
	applyStringEnv(&c.Gitea.Username, envPrefix+"GITEA_USERNAME")
	applyCSVEnv(&c.Gitea.Orgs, envPrefix+"GITEA_ORGS")

	applyStringEnv(&c.Mirror.Dir, envPrefix+"MIRROR_DIR")
	applyStringEnv(&c.Mirror.Email, envPrefix+"MIRROR_EMAIL")
	applyStringEnv(&c.Mirror.Remote, envPrefix+"MIRROR_REMOTE")

	applyStringEnv(&c.Sync.Since, envPrefix+"SYNC_SINCE")
	applyCSVEnv(&c.Sync.ActivityTypes, envPrefix+"SYNC_ACTIVITY_TYPES")
	applyBoolEnv(&c.Sync.CopyMessages, envPrefix+"SYNC_COPY_MESSAGES")
	applyCSVEnv(&c.Sync.ExcludeRepos, envPrefix+"SYNC_EXCLUDE_REPOS")
	applyBoolEnv(&c.Sync.ExcludeForks, envPrefix+"SYNC_EXCLUDE_FORKS")
	applyBoolEnv(&c.Sync.ExcludeArchived, envPrefix+"SYNC_EXCLUDE_ARCHIVED")
}

func (c *Config) normalize() {
	c.Gitea.URL = strings.TrimRight(strings.TrimSpace(c.Gitea.URL), "/")
	c.Gitea.Token = strings.TrimSpace(c.Gitea.Token)
	c.Gitea.Username = strings.TrimSpace(c.Gitea.Username)
	c.Gitea.Orgs = normalizeList(c.Gitea.Orgs)

	c.Mirror.Dir = expandHome(strings.TrimSpace(c.Mirror.Dir))
	c.Mirror.Email = strings.TrimSpace(c.Mirror.Email)
	c.Mirror.Remote = strings.TrimSpace(c.Mirror.Remote)

	c.Sync.Since = strings.TrimSpace(c.Sync.Since)
	c.Sync.ActivityTypes = normalizeUniqueList(c.Sync.ActivityTypes)
	c.Sync.ExcludeRepos = normalizeUniqueList(c.Sync.ExcludeRepos)
}

func parseConfig(data []byte, cfg *Config) error {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	currentSection := ""
	currentList := ""

	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))

		switch {
		case indent == 0 && strings.HasSuffix(trimmed, ":"):
			currentSection = strings.TrimSuffix(trimmed, ":")
			currentList = ""
		case indent == 2 && strings.HasPrefix(trimmed, "- "):
			return fmt.Errorf("line %d: list item without list key", lineNumber)
		case indent == 2:
			key, value, ok := splitKeyValue(trimmed)
			if !ok {
				return fmt.Errorf("line %d: invalid key/value", lineNumber)
			}
			if currentSection == "" {
				return fmt.Errorf("line %d: nested key without section", lineNumber)
			}
			if value == "" {
				currentList = currentSection + "." + key
				assignList(cfg, currentList, nil)
				continue
			}
			currentList = ""
			if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
				assignList(cfg, currentSection+"."+key, parseInlineList(value))
				continue
			}
			if err := assignScalar(cfg, currentSection, key, unquote(value)); err != nil {
				return fmt.Errorf("line %d: %w", lineNumber, err)
			}
		case indent == 4 && strings.HasPrefix(trimmed, "- "):
			if currentList == "" {
				return fmt.Errorf("line %d: list item without active list", lineNumber)
			}
			assignListItem(cfg, currentList, unquote(strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))))
		default:
			return fmt.Errorf("line %d: unsupported indentation or syntax", lineNumber)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func splitKeyValue(line string) (string, string, bool) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

func assignScalar(cfg *Config, section, key, value string) error {
	switch section + "." + key {
	case "gitea.url":
		cfg.Gitea.URL = value
	case "gitea.token":
		cfg.Gitea.Token = value
	case "gitea.username":
		cfg.Gitea.Username = value
	case "mirror.dir":
		cfg.Mirror.Dir = value
	case "mirror.email":
		cfg.Mirror.Email = value
	case "mirror.remote":
		cfg.Mirror.Remote = value
	case "sync.since":
		cfg.Sync.Since = value
	case "sync.copy_messages":
		cfg.Sync.CopyMessages = parseBool(value)
	case "sync.exclude_forks":
		cfg.Sync.ExcludeForks = parseBool(value)
	case "sync.exclude_archived":
		cfg.Sync.ExcludeArchived = parseBool(value)
	default:
		return fmt.Errorf("unknown config key %s.%s", section, key)
	}
	return nil
}

func assignList(cfg *Config, target string, values []string) {
	switch target {
	case "gitea.orgs":
		cfg.Gitea.Orgs = values
	case "sync.activity_types":
		cfg.Sync.ActivityTypes = values
	case "sync.exclude_repos":
		cfg.Sync.ExcludeRepos = values
	}
}

func assignListItem(cfg *Config, target, value string) {
	switch target {
	case "gitea.orgs":
		cfg.Gitea.Orgs = append(cfg.Gitea.Orgs, value)
	case "sync.activity_types":
		cfg.Sync.ActivityTypes = append(cfg.Sync.ActivityTypes, value)
	case "sync.exclude_repos":
		cfg.Sync.ExcludeRepos = append(cfg.Sync.ExcludeRepos, value)
	}
}

func parseInlineList(value string) []string {
	trimmed := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		result = append(result, unquote(strings.TrimSpace(part)))
	}
	return result
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	value = strings.Trim(value, `'`)
	return value
}

func parseBool(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == "true" || value == "1" || value == "yes"
}

func applyStringEnv(dst *string, key string) {
	if value, ok := os.LookupEnv(key); ok {
		*dst = value
	}
}

func applyCSVEnv(dst *[]string, key string) {
	if value, ok := os.LookupEnv(key); ok {
		*dst = strings.Split(value, ",")
	}
}

func applyBoolEnv(dst *bool, key string) {
	if value, ok := os.LookupEnv(key); ok {
		*dst = parseBool(value)
	}
}

func normalizeList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func normalizeUniqueList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range normalizeList(values) {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func expandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/"))
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}
