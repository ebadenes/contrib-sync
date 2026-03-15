package report

import (
	"strings"
	"testing"
	"time"

	"github.com/ebadenes/contrib-sync/internal/activity"
	"github.com/ebadenes/contrib-sync/internal/gitea"
)

func TestRenderSyncSummaryIncludesKeyMetrics(t *testing.T) {
	summary := SyncSummary{
		ConfigPath:      "config.yaml",
		MirrorDir:       "/tmp/mirror",
		RepositoryCount: 2,
		CollectedCount:  5,
		ExistingCount:   2,
		PendingCount:    3,
		CreatedCount:    3,
		CountsByType: map[string]int{
			activity.TypeCommit: 2,
			activity.TypePR:     1,
			activity.TypeIssue:  1,
			activity.TypeReview: 1,
		},
		Repositories: []gitea.Repository{{Name: "demo", Owner: gitea.Owner{Login: "alice"}}},
		PendingPreview: []activity.Event{{Type: activity.TypeCommit, Repository: "alice/demo", Timestamp: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)}},
	}

	output := RenderSyncSummary(summary)
	for _, expected := range []string{
		"loaded config from config.yaml",
		"discovered 2 repositories after filtering",
		"created mirror commits: 3",
		"- commits: 2",
		"- alice/demo",
		"first pending mirror events:",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected summary to contain %q\noutput:\n%s", expected, output)
		}
	}
}

func TestRenderMirrorREADMEIncludesRepositoriesAndBreakdown(t *testing.T) {
	summary := SyncSummary{
		RepositoryCount: 1,
		CollectedCount:  4,
		ExistingCount:   1,
		PendingCount:    3,
		CreatedCount:    3,
		GeneratedAt:     time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC),
		CountsByType: map[string]int{
			activity.TypeCommit: 2,
			activity.TypePR:     1,
			activity.TypeIssue:  1,
			activity.TypeReview: 0,
		},
		Repositories: []gitea.Repository{{Name: "demo", Owner: gitea.Owner{Login: "alice"}}},
		PendingPreview: []activity.Event{{Type: activity.TypeIssue, Repository: "alice/demo", Timestamp: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)}},
	}

	output := RenderMirrorREADME(summary)
	for _, expected := range []string{
		"# Contribution Mirror",
		"- Repositories scanned: `1`",
		"- commits: 2",
		"- `alice/demo`",
		"## Recent Pending Events Preview",
		"Generated automatically by `contrib-sync`.",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected readme to contain %q\noutput:\n%s", expected, output)
		}
	}
}
