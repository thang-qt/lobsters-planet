package feeds

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"lobsters-planet/internal/state"
)

func Refresh(ctx context.Context, client *http.Client, feed state.DiscoveredFeed, userAgent string) (string, []state.FeedEntry, error) {
	parser := gofeed.NewParser()
	parser.Client = client
	parser.UserAgent = userAgent

	parsed, err := parser.ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		return "", nil, fmt.Errorf("parse feed %s: %w", feed.URL, err)
	}

	entries := make([]state.FeedEntry, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		entryID := entryID(feed.URL, item)
		if entryID == "" {
			continue
		}
		entries = append(entries, state.FeedEntry{
			ID:            entryID,
			Title:         strings.TrimSpace(item.Title),
			URL:           strings.TrimSpace(item.Link),
			Summary:       summary(item),
			PublishedAt:   normalizeTime(item.PublishedParsed),
			UpdatedAt:     normalizeTime(item.UpdatedParsed),
			FeedURL:       feed.URL,
			FeedTitle:     strings.TrimSpace(parsed.Title),
			OwnerUsername: feed.OwnerUsername,
			SiteURL:       feed.SiteURL,
		})
	}
	return strings.TrimSpace(parsed.Title), entries, nil
}

func entryID(feedURL string, item *gofeed.Item) string {
	for _, candidate := range []string{item.GUID, item.Link} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}
	if item.Title != "" && item.Published != "" {
		return feedURL + "#" + item.Published + ":" + item.Title
	}
	return ""
}

func summary(item *gofeed.Item) string {
	for _, candidate := range []string{item.Description, item.Content} {
		candidate = strings.Join(strings.Fields(candidate), " ")
		if len(candidate) > 500 {
			return candidate[:500] + "…"
		}
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func normalizeTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}
