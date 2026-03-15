package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ebadenes/contrib-sync/internal/activity"
	"github.com/ebadenes/contrib-sync/internal/config"
	"github.com/ebadenes/contrib-sync/internal/gitea"
	"github.com/ebadenes/contrib-sync/internal/mirror"
	"github.com/ebadenes/contrib-sync/internal/report"
)

const version = "0.1.0"

func printUsage() {
	fmt.Fprintf(os.Stdout, "contrib-sync %s\n\n", version)
	fmt.Fprintln(os.Stdout, "A CLI to mirror contribution timestamps from Gitea into a GitHub mirror repository.")
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, "Usage:")
	fmt.Fprintln(os.Stdout, "  contrib-sync sync [--config path]")
	fmt.Fprintln(os.Stdout, "  contrib-sync init [--config path]")
	fmt.Fprintln(os.Stdout, "  contrib-sync status [--config path]")
	fmt.Fprintln(os.Stdout, "  contrib-sync version")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "sync":
		runSync(os.Args[2:])
	case "init":
		runInit(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "version":
		fmt.Fprintln(os.Stdout, version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runSync(args []string) {
	cfg, path := mustLoadConfig(args, "sync")
	client := newGiteaClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repos, err := discoverRepositories(ctx, client, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: discover repositories: %v\n", err)
		os.Exit(1)
	}

	events, err := activity.Collect(ctx, client, repos, activity.CollectOptions{
		Username:      cfg.Gitea.Username,
		Since:         cfg.Sync.Since,
		ActivityTypes: cfg.Sync.ActivityTypes,
		CopyMessages:  cfg.Sync.CopyMessages,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: collect activity: %v\n", err)
		os.Exit(1)
	}

	mirrorRepo := mirror.NewRepository(cfg.Mirror.Dir, cfg.Mirror.Email)
	existingTimestamps, err := mirrorRepo.ExistingTimestamps(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: inspect mirror repository: %v\n", err)
		os.Exit(1)
	}
	pendingEvents := activity.ExcludeTimestamps(events, existingTimestamps)
	created, err := mirrorRepo.WriteEvents(ctx, pendingEvents)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: write mirror commits: %v\n", err)
		os.Exit(1)
	}

	summary := report.SyncSummary{
		ConfigPath:      path,
		MirrorDir:       cfg.Mirror.Dir,
		RepositoryCount: len(repos),
		CollectedCount:  len(events),
		ExistingCount:   len(existingTimestamps),
		PendingCount:    len(pendingEvents),
		CreatedCount:    created,
		CountsByType:    countsWithDefaults(activity.CountByType(events)),
		Repositories:    repos,
		PendingPreview:  report.PreviewEvents(pendingEvents, 10),
		GeneratedAt:     time.Now().UTC(),
	}

	if err := mirrorRepo.WriteFile("README.md", report.RenderMirrorREADME(summary)); err != nil {
		fmt.Fprintf(os.Stderr, "sync: write mirror readme: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprint(os.Stdout, report.RenderSyncSummary(summary))
}

func runStatus(args []string) {
	cfg, path := mustLoadConfig(args, "status")
	client := newGiteaClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Fprintf(os.Stdout, "config file: %s\n\n", path)
	fmt.Fprintln(os.Stdout, cfg.ConfigSummary())
	fmt.Fprintln(os.Stdout, "")

	version, err := client.Version(ctx)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Gitea connectivity: error (%v)\n", err)
	} else {
		fmt.Fprintf(os.Stdout, "Gitea connectivity: ok (version %s)\n", version.Version)
	}

	repos, repoErr := discoverRepositories(ctx, client, cfg)
	if repoErr != nil {
		fmt.Fprintf(os.Stdout, "Repository discovery: error (%v)\n", repoErr)
	} else {
		fmt.Fprintf(os.Stdout, "Repository discovery: ok (%d repositories after filtering)\n", len(repos))
	}

	fmt.Fprintf(os.Stdout, "Mirror directory: %s\n", mirrorStatus(cfg.Mirror.Dir))
	fmt.Fprintf(os.Stdout, "Git binary: %s\n", gitStatus())
}

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	configPath := fs.String("config", config.DefaultConfigPath(), "Path to the YAML configuration file")
	_ = fs.Parse(args)

	if _, err := os.Stat(*configPath); err == nil {
		fmt.Fprintf(os.Stderr, "config file already exists: %s\n", *configPath)
		os.Exit(1)
	}

	if err := os.WriteFile(*configPath, []byte(config.ExampleConfig()), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "write config file %s: %v\n", *configPath, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "created config file at %s\n", *configPath)
}

func mustLoadConfig(args []string, name string) (*config.Config, string) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	configPath := fs.String("config", config.DefaultConfigPath(), "Path to the YAML configuration file")
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		os.Exit(1)
	}

	return cfg, *configPath
}

func newGiteaClient(cfg *config.Config) *gitea.Client {
	return gitea.NewClient(gitea.NewClientOptions{
		BaseURL:   cfg.Gitea.URL,
		Token:     cfg.Gitea.Token,
		UserAgent: "contrib-sync/" + version,
	})
}

func discoverRepositories(ctx context.Context, client *gitea.Client, cfg *config.Config) ([]gitea.Repository, error) {
	repos, err := client.ListUserRepos(ctx, cfg.Gitea.Username)
	if err != nil {
		return nil, err
	}

	for _, org := range cfg.Gitea.Orgs {
		orgRepos, err := client.ListOrgRepos(ctx, org)
		if err != nil {
			return nil, err
		}
		repos = append(repos, orgRepos...)
	}

	return client.FilterRepositories(repos, cfg.Sync.ExcludeForks, cfg.Sync.ExcludeArchived, cfg.Sync.ExcludeRepos), nil
}

func mirrorStatus(dir string) string {
	info, err := os.Stat(dir)
	if err != nil {
		return "missing"
	}
	if !info.IsDir() {
		return "not a directory"
	}
	if _, err := os.Stat(strings.TrimRight(dir, "/") + "/.git"); err == nil {
		return "present (git repo)"
	}
	return "present (directory only)"
}

func gitStatus() string {
	if _, err := exec.LookPath("git"); err != nil {
		return "missing"
	}
	return "available"
}

func countsWithDefaults(counts map[string]int) map[string]int {
	result := map[string]int{
		activity.TypeCommit: 0,
		activity.TypePR:     0,
		activity.TypeIssue:  0,
		activity.TypeReview: 0,
	}
	for key, value := range counts {
		result[key] = value
	}
	return result
}
