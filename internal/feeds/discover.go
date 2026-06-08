package feeds

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

type DiscoveryResult struct {
	FeedURLs []string
}

func Discover(ctx context.Context, client *http.Client, homepageURL, userAgent string, commonPaths []string) (DiscoveryResult, error) {
	homepage, body, err := fetch(ctx, client, homepageURL, userAgent)
	if err != nil {
		return DiscoveryResult{}, err
	}

	candidates := feedLinks(homepage, body)
	if len(candidates) == 0 {
		candidates = commonFeedURLs(homepage, commonPaths)
	}

	valid := make(map[string]bool)
	for _, candidate := range candidates {
		if valid[candidate] {
			continue
		}
		if ok := looksLikeFeed(ctx, client, candidate, userAgent); ok {
			valid[candidate] = true
		}
	}

	urls := make([]string, 0, len(valid))
	for feedURL := range valid {
		urls = append(urls, feedURL)
	}
	sort.Strings(urls)
	return DiscoveryResult{FeedURLs: urls}, nil
}

func fetch(ctx context.Context, client *http.Client, pageURL, userAgent string) (*url.URL, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch %s: %w", pageURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("fetch %s: unexpected status %s", pageURL, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2_000_000))
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", pageURL, err)
	}
	return resp.Request.URL, body, nil
}

func feedLinks(base *url.URL, body []byte) []string {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var links []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" && isAlternateFeedLink(n) {
			if href := attr(n, "href"); href != "" {
				if resolved, err := base.Parse(href); err == nil && isHTTP(resolved) && !seen[resolved.String()] {
					seen[resolved.String()] = true
					links = append(links, resolved.String())
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return links
}

func isAlternateFeedLink(n *html.Node) bool {
	rel := strings.ToLower(attr(n, "rel"))
	if !strings.Contains(rel, "alternate") {
		return false
	}
	typeValue := strings.ToLower(attr(n, "type"))
	return strings.Contains(typeValue, "rss") || strings.Contains(typeValue, "atom") || strings.Contains(typeValue, "feed+json") || strings.Contains(typeValue, "jsonfeed")
}

func commonFeedURLs(base *url.URL, paths []string) []string {
	if len(paths) == 0 {
		paths = []string{"/feed.xml", "/feed", "/rss.xml", "/rss", "/atom.xml", "/index.xml"}
	}
	urls := make([]string, 0, len(paths))
	for _, path := range paths {
		resolved := *base
		resolved.RawQuery = ""
		resolved.Fragment = ""
		if strings.HasPrefix(path, "/") {
			resolved.Path = path
		} else {
			resolved.Path = "/" + path
		}
		urls = append(urls, resolved.String())
	}
	return urls
}

func looksLikeFeed(ctx context.Context, client *http.Client, feedURL, userAgent string) bool {
	_, body, err := fetch(ctx, client, feedURL, userAgent)
	if err != nil {
		return false
	}
	prefix := strings.ToLower(strings.TrimSpace(string(body[:min(len(body), 4096)])))
	return strings.Contains(prefix, "<rss") || strings.Contains(prefix, "<feed") || strings.Contains(prefix, "<rdf:rdf") || strings.Contains(prefix, "\"version\":\"https://jsonfeed.org/version/") || strings.Contains(prefix, "\"version\": \"https://jsonfeed.org/version/")
}

func attr(n *html.Node, name string) string {
	for _, attr := range n.Attr {
		if strings.EqualFold(attr.Key, name) {
			return attr.Val
		}
	}
	return ""
}

func isHTTP(u *url.URL) bool {
	return u.Scheme == "http" || u.Scheme == "https"
}
