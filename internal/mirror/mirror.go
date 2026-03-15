package mirror

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ebadenes/contrib-sync/internal/activity"
)

const defaultAuthorName = "contrib-sync"

type Repository struct {
	Dir   string
	Email string
}

func NewRepository(dir, email string) *Repository {
	return &Repository{
		Dir:   strings.TrimSpace(dir),
		Email: strings.TrimSpace(email),
	}
}

func (r *Repository) Ensure(ctx context.Context) error {
	if strings.TrimSpace(r.Dir) == "" {
		return errors.New("mirror directory is required")
	}

	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		return fmt.Errorf("create mirror directory %s: %w", r.Dir, err)
	}

	gitDir := filepath.Join(r.Dir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat mirror git directory %s: %w", gitDir, err)
	}

	if _, err := r.runGit(ctx, nil, "init", "--initial-branch=main"); err != nil {
		return fmt.Errorf("initialize mirror repository: %w", err)
	}
	return nil
}

func (r *Repository) ExistingTimestamps(ctx context.Context) ([]time.Time, error) {
	if strings.TrimSpace(r.Dir) == "" {
		return nil, errors.New("mirror directory is required")
	}

	gitDir := filepath.Join(r.Dir, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat mirror git directory %s: %w", gitDir, err)
	}

	output, err := r.runGit(ctx, nil, "log", "--pretty=format:%aI")
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits yet") {
			return nil, nil
		}
		return nil, fmt.Errorf("read mirror timestamps: %w", err)
	}

	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil, nil
	}

	lines := strings.Split(trimmed, "\n")
	timestamps := make([]time.Time, 0, len(lines))
	for _, line := range lines {
		parsed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(line))
		if parseErr != nil {
			return nil, fmt.Errorf("parse mirror timestamp %q: %w", line, parseErr)
		}
		timestamps = append(timestamps, parsed.UTC())
	}
	return timestamps, nil
}

func (r *Repository) WriteEvents(ctx context.Context, events []activity.Event) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	if strings.TrimSpace(r.Email) == "" {
		return 0, errors.New("mirror email is required")
	}
	if err := r.Ensure(ctx); err != nil {
		return 0, err
	}

	ordered := append([]activity.Event(nil), events...)
	activity.Sort(ordered)

	created := 0
	for _, event := range ordered {
		env := []string{
			"GIT_AUTHOR_NAME=" + defaultAuthorName,
			"GIT_AUTHOR_EMAIL=" + r.Email,
			"GIT_AUTHOR_DATE=" + event.Timestamp.Format(time.RFC3339),
			"GIT_COMMITTER_NAME=" + defaultAuthorName,
			"GIT_COMMITTER_EMAIL=" + r.Email,
			"GIT_COMMITTER_DATE=" + event.Timestamp.Format(time.RFC3339),
		}

		message := strings.TrimSpace(event.Message)
		if message == "" {
			message = fmt.Sprintf("mirror %s activity from %s", event.Type, event.Repository)
		}

		if _, err := r.runGit(ctx, env, "commit", "--allow-empty", "-m", message); err != nil {
			return created, fmt.Errorf("create mirror commit for %s at %s: %w", event.Repository, event.Timestamp.Format(time.RFC3339), err)
		}
		created++
	}

	return created, nil
}

func (r *Repository) WriteFile(name, content string) error {
	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		return fmt.Errorf("create mirror directory %s: %w", r.Dir, err)
	}
	path := filepath.Join(r.Dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write mirror file %s: %w", path, err)
	}
	return nil
}

func (r *Repository) runGit(ctx context.Context, extraEnv []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(), extraEnv...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
