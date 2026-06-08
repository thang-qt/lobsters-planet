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
