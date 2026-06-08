package feeds

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"

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
		candidate = plainText(candidate)
		if len(candidate) > 500 {
			return candidate[:500] + "…"
		}
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func plainText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "<") || !strings.Contains(value, ">") {
		return strings.Join(strings.Fields(value), " ")
	}

	var builder strings.Builder
	tokenizer := html.NewTokenizer(bytes.NewBufferString(value))
	skipDepth := 0
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				return strings.Join(strings.Fields(builder.String()), " ")
			}
			return strings.Join(strings.Fields(value), " ")
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			name := strings.ToLower(token.Data)
			if name == "script" || name == "style" || name == "noscript" {
				skipDepth++
				continue
			}
			if skipDepth == 0 && (name == "p" || name == "br" || name == "li" || name == "div" || name == "blockquote" || name == "h1" || name == "h2" || name == "h3") {
				builder.WriteByte(' ')
			}
		case html.EndTagToken:
			token := tokenizer.Token()
			name := strings.ToLower(token.Data)
			if name == "script" || name == "style" || name == "noscript" {
				if skipDepth > 0 {
					skipDepth--
				}
				continue
			}
			if skipDepth == 0 && (name == "p" || name == "li" || name == "div" || name == "blockquote") {
				builder.WriteByte(' ')
			}
		case html.TextToken:
			if skipDepth == 0 {
				builder.Write(tokenizer.Text())
				builder.WriteByte(' ')
			}
		}
	}
}

func normalizeTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}
