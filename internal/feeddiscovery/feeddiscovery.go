package feeddiscovery

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"lobsters-planet/internal/config"
	"lobsters-planet/internal/feeds"
	"lobsters-planet/internal/state"
)

type Result struct {
	SitesSelected  int
	SitesSucceeded int
	SitesFailed    int
	FeedsFound     int
}

func Run(ctx context.Context, cfg config.Config) (Result, error) {
	store, err := state.Load(cfg.Output.StateFile)
	if err != nil {
		return Result{}, err
	}

	client := &http.Client{Timeout: time.Duration(cfg.Feeds.RequestTimeoutSeconds) * time.Second}
	now := time.Now().UTC()
	due := store.SitesDueForFeedDiscovery(now, time.Duration(cfg.Feeds.DiscoveryRecheckDays)*24*time.Hour, cfg.Feeds.MaxSitesPerDiscoveryRun)

	result := Result{SitesSelected: len(due)}
	for index, site := range due {
		if index > 0 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(time.Duration(cfg.Feeds.RequestDelaySeconds) * time.Second):
			}
		}

		discovered, err := feeds.Discover(ctx, client, site.HomepageURL, cfg.UserAgent, cfg.Feeds.CommonPaths)
		checkedAt := time.Now().UTC()
		if err != nil {
			store.RecordFeedDiscoveryFailure(site.Username, checkedAt, err)
			result.SitesFailed++
			fmt.Printf("site %s failed: %v\n", site.Username, err)
			continue
		}

		store.RecordFeedDiscoverySuccess(site.Username, discovered.FeedURLs, checkedAt)
		result.SitesSucceeded++
		result.FeedsFound += len(discovered.FeedURLs)
		if len(discovered.FeedURLs) == 0 {
			fmt.Printf("site %s no feeds\n", site.Username)
		} else {
			fmt.Printf("site %s feeds %v\n", site.Username, discovered.FeedURLs)
		}
	}

	if err := state.Save(cfg.Output.StateFile, store); err != nil {
		return result, err
	}
	return result, nil
}
