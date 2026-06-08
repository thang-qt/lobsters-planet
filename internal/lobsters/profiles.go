package lobsters

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type Profile struct {
	Username         string
	ProfileURL       string
	HomepageURL      string
	JoinedAt         *time.Time
	Karma            *int
	StoriesSubmitted *int
	CommentsPosted   *int
	About            string
}

func FetchProfile(ctx context.Context, client *http.Client, profileURL, userAgent string) (Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, profileURL, nil)
	if err != nil {
		return Profile{}, fmt.Errorf("create profile request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html")

	resp, err := client.Do(req)
	if err != nil {
		return Profile{}, fmt.Errorf("fetch profile: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Profile{}, fmt.Errorf("fetch profile: unexpected status %s", resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return Profile{}, fmt.Errorf("parse profile page: %w", err)
	}

	base, err := url.Parse(profileURL)
	if err != nil {
		return Profile{}, fmt.Errorf("parse profile URL: %w", err)
	}

	profile := Profile{ProfileURL: profileURL, Username: usernameFromProfileURL(profileURL)}
	fields := profileFields(doc)
	profile.HomepageURL = firstHTTPLink(fields["Homepage"], base)
	profile.JoinedAt = joinedAt(fields["Joined"])
	profile.Karma = intField(fields["Karma"])
	profile.StoriesSubmitted = intField(fields["Stories Submitted"])
	profile.CommentsPosted = intField(fields["Comments Posted"])
	profile.About = cleanText(textContent(fields["About"]))
	return profile, nil
}

func usernameFromProfileURL(profileURL string) string {
	parsed, err := url.Parse(profileURL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.Trim(parsed.Path, "/"), "~")
}

func profileFields(doc *html.Node) map[string]*html.Node {
	fields := make(map[string]*html.Node)
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "label" {
			label := cleanText(textContent(n))
			if label != "" {
				for sibling := n.NextSibling; sibling != nil; sibling = sibling.NextSibling {
					if sibling.Type == html.ElementNode && (sibling.Data == "span" || sibling.Data == "div") {
						fields[label] = sibling
						break
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return fields
}

func joinedAt(n *html.Node) *time.Time {
	if n == nil {
		return nil
	}
	var found *time.Time
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "time" {
			for _, attr := range n.Attr {
				if attr.Key == "datetime" || attr.Key == "title" {
					if parsed := parseLobstersTime(attr.Val); parsed != nil {
						found = parsed
						return
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return found
}

func parseLobstersTime(value string) *time.Time {
	layouts := []string{"2006-01-02 15:04:05", time.RFC3339}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, strings.TrimSpace(value))
		if err == nil {
			parsed = parsed.UTC()
			return &parsed
		}
	}
	return nil
}

func intField(n *html.Node) *int {
	if n == nil {
		return nil
	}
	text := cleanText(textContent(n))
	for _, token := range strings.Fields(text) {
		token = strings.Trim(token, "(),")
		if value, err := strconv.Atoi(token); err == nil {
			return &value
		}
	}
	return nil
}

func firstHTTPLink(n *html.Node, base *url.URL) string {
	if n == nil {
		return ""
	}
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key != "href" {
				continue
			}
			resolved, err := base.Parse(attr.Val)
			if err != nil {
				return ""
			}
			if resolved.Scheme == "http" || resolved.Scheme == "https" {
				return resolved.String()
			}
		}
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if link := firstHTTPLink(child, base); link != "" {
			return link
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			builder.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return builder.String()
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
