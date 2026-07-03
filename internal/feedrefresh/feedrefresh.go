package feedrefresh

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
	FeedsSelected  int
	FeedsSucceeded int
	FeedsFailed    int
	EntriesFound   int
}

func Run(ctx context.Context, cfg config.Config) (Result, error) {
	store, err := state.Load(cfg.Output.StateFile)
	if err != nil {
		return Result{}, err
	}

	client := &http.Client{Timeout: time.Duration(cfg.Feeds.RequestTimeoutSeconds) * time.Second}
	now := time.Now().UTC()
	due := store.FeedsDueForRefresh(now, time.Duration(cfg.Feeds.RefreshIntervalHours)*time.Hour, cfg.Feeds.MaxFeedsPerRefreshRun)
	due = filterFeeds(due, excludedSet(cfg.Feeds.ExcludeSiteURLs), excludedSet(cfg.Feeds.ExcludeFeedURLs), cfg.Feeds.ExcludeSitePatterns)

	result := Result{FeedsSelected: len(due)}
	for index, feed := range due {
		if index > 0 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(time.Duration(cfg.Feeds.RequestDelaySeconds) * time.Second):
			}
		}

		title, entries, err := feeds.Refresh(ctx, client, feed, cfg.UserAgent)
		checkedAt := time.Now().UTC()
		if err != nil {
			store.RecordFeedRefreshFailure(feed.URL, checkedAt, err)
			result.FeedsFailed++
			fmt.Printf("feed %s failed: %v\n", feed.URL, err)
			continue
		}

		store.RecordFeedRefreshSuccess(feed.URL, title, entries, checkedAt)
		result.FeedsSucceeded++
		result.EntriesFound += len(entries)
		fmt.Printf("feed %s entries %d\n", feed.URL, len(entries))
	}

	if err := state.Save(cfg.Output.StateFile, store); err != nil {
		return result, err
	}
	return result, nil
}

func excludedSet(values []string) map[string]bool {
	excluded := make(map[string]bool)
	for _, value := range values {
		excluded[config.NormalizeURL(value)] = true
	}
	return excluded
}

func filterFeeds(feeds []state.DiscoveredFeed, excludedSites, excludedFeeds map[string]bool, patterns []string) []state.DiscoveredFeed {
	if len(excludedSites) == 0 && len(excludedFeeds) == 0 && len(patterns) == 0 {
		return feeds
	}
	filtered := make([]state.DiscoveredFeed, 0, len(feeds))
	for _, feed := range feeds {
		if excludedFeeds[config.NormalizeURL(feed.URL)] || excludedSites[config.NormalizeURL(feed.SiteURL)] {
			continue
		}
		if config.MatchSitePatterns(feed.SiteURL, patterns) {
			continue
		}
		filtered = append(filtered, feed)
	}
	return filtered
}
