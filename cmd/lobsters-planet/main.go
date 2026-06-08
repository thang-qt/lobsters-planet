package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"lobsters-planet/internal/config"
	"lobsters-planet/internal/discovery"
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

	result, _, err := discovery.Run(context.Background(), cfg)
	if err != nil {
		return err
	}

	fmt.Printf("fetched %d Lobste.rs users\n", result.FetchedUsers)
	fmt.Printf("new users: %d\n", result.NewUsers)
	fmt.Printf("known users: %d\n", result.KnownUsers)
	fmt.Printf("profiles selected: %d\n", result.ProfilesSelected)
	fmt.Printf("profiles succeeded: %d\n", result.ProfilesSucceeded)
	fmt.Printf("profiles failed: %d\n", result.ProfilesFailed)
	fmt.Printf("homepages found: %d\n", result.HomepagesFound)
	fmt.Printf("wrote %s\n", cfg.Output.StateFile)
	return nil
}
