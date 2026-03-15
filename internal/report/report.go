package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ebadenes/contrib-sync/internal/activity"
	"github.com/ebadenes/contrib-sync/internal/gitea"
)

type SyncSummary struct {
	ConfigPath          string
	MirrorDir           string
	RepositoryCount     int
	CollectedCount      int
	ExistingCount       int
	PendingCount        int
	CreatedCount        int
	CountsByType        map[string]int
	Repositories        []gitea.Repository
	PendingPreview      []activity.Event
	GeneratedAt         time.Time
}

func RenderSyncSummary(summary SyncSummary) string {
	lines := []string{
		fmt.Sprintf("loaded config from %s", summary.ConfigPath),
		fmt.Sprintf("discovered %d repositories after filtering", summary.RepositoryCount),
		fmt.Sprintf("collected %d normalized activity events", summary.CollectedCount),
		fmt.Sprintf("mirror already had %d timestamps", summary.ExistingCount),
		fmt.Sprintf("pending mirror events: %d", summary.PendingCount),
		fmt.Sprintf("created mirror commits: %d", summary.CreatedCount),
	}

	for _, line := range renderCountLines(summary.CountsByType) {
		lines = append(lines, line)
	}

	lines = append(lines, "")
	for _, repo := range summary.Repositories {
		lines = append(lines, fmt.Sprintf("- %s/%s", repo.Owner.Login, repo.Name))
	}

	if len(summary.PendingPreview) > 0 {
		lines = append(lines, "", "first pending mirror events:")
		for _, event := range summary.PendingPreview {
			lines = append(lines, fmt.Sprintf("- %s | %s | %s", event.Timestamp.Format(time.RFC3339), event.Type, event.Repository))
		}
	}

	lines = append(lines, "", fmt.Sprintf("mirror directory: %s", summary.MirrorDir))
	return strings.Join(lines, "\n") + "\n"
}

func RenderMirrorREADME(summary SyncSummary) string {
	generatedAt := summary.GeneratedAt.UTC()
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	lines := []string{
		"# Contribution Mirror",
		"",
		"This repository is maintained by `contrib-sync`.",
		"It mirrors contribution timestamps from Gitea into Git commits so the contribution graph can reflect work done outside GitHub.",
		"",
		"## Last Sync",
		"",
		fmt.Sprintf("- Generated at: `%s`", generatedAt.Format(time.RFC3339)),
		fmt.Sprintf("- Repositories scanned: `%d`", summary.RepositoryCount),
		fmt.Sprintf("- Events collected: `%d`", summary.CollectedCount),
		fmt.Sprintf("- Existing mirror timestamps: `%d`", summary.ExistingCount),
		fmt.Sprintf("- Pending events: `%d`", summary.PendingCount),
		fmt.Sprintf("- Commits created in this sync: `%d`", summary.CreatedCount),
		"",
		"## Activity Breakdown",
		"",
	}

	for _, line := range renderCountLines(summary.CountsByType) {
		lines = append(lines, strings.Replace(line, "- ", "- ", 1))
	}

	lines = append(lines, "", "## Repositories", "")
	for _, repo := range summary.Repositories {
		lines = append(lines, fmt.Sprintf("- `%s/%s`", repo.Owner.Login, repo.Name))
	}

	if len(summary.PendingPreview) > 0 {
		lines = append(lines, "", "## Recent Pending Events Preview", "")
		for _, event := range summary.PendingPreview {
			lines = append(lines, fmt.Sprintf("- `%s` `%s` `%s`", event.Timestamp.Format(time.RFC3339), event.Type, event.Repository))
		}
	}

	lines = append(lines, "", "Generated automatically by `contrib-sync`.", "")
	return strings.Join(lines, "\n")
}

func PreviewEvents(events []activity.Event, limit int) []activity.Event {
	if len(events) <= limit {
		return append([]activity.Event(nil), events...)
	}
	return append([]activity.Event(nil), events[:limit]...)
}

func renderCountLines(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("- %s: %d", key, counts[key]))
	}
	return lines
}
