package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"lobsters-planet/internal/config"
	"lobsters-planet/internal/lobsters"
	"lobsters-planet/internal/state"
)

const usage = `lobsters-planet builds a static planet from Lobste.rs users' personal sites.

Usage:
  lobsters-planet [--config config.yaml] <command>

Commands:
  discover  Discover Lobste.rs users and update local crawler state
  refresh   Refresh known feeds
  build     Generate the static site and combined feed
  help      Show this help
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("lobsters-planet", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("config", "config.yaml", "path to config file")
	if err := flags.Parse(args); err != nil {
		return err
	}

	remaining := flags.Args()
	if len(remaining) < 1 {
		fmt.Print(usage)
		return nil
	}

	switch remaining[0] {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "discover":
		return discover(*configPath)
	case "refresh", "build":
		return fmt.Errorf("%s is not implemented yet", remaining[0])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", remaining[0])
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("unknown command")
	}
}

func discover(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	store, err := state.Load(cfg.Output.StateFile)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: time.Duration(cfg.Lobsters.RequestTimeoutSeconds) * time.Second}
	ctx := context.Background()
	checkedAt := time.Now().UTC()

	users, err := lobsters.FetchUsers(ctx, client, cfg.Lobsters.UsersURL, cfg.UserAgent)
	if err != nil {
		return err
	}
	newUsers := store.MergeUsers(users, checkedAt)

	if err := state.Save(cfg.Output.StateFile, store); err != nil {
		return err
	}

	fmt.Printf("fetched %d Lobste.rs users\n", len(users))
	fmt.Printf("new users: %d\n", newUsers)
	fmt.Printf("known users: %d\n", len(store.Users))
	fmt.Printf("wrote %s\n", cfg.Output.StateFile)
	return nil
}
