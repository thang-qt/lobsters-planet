package lobsters

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

type User struct {
	Username   string `json:"username"`
	ProfileURL string `json:"profile_url"`
}

func FetchUsers(ctx context.Context, client *http.Client, usersURL, userAgent string) ([]User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usersURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create users request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch users: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch users: unexpected status %s", resp.Status)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse users page: %w", err)
	}
	base, err := url.Parse(usersURL)
	if err != nil {
		return nil, fmt.Errorf("parse users URL: %w", err)
	}

	usersByName := make(map[string]User)
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key != "href" || !strings.HasPrefix(attr.Val, "/~") {
					continue
				}
				username := strings.TrimPrefix(attr.Val, "/~")
				if username == "" || strings.Contains(username, "/") {
					continue
				}
				profile, err := base.Parse(attr.Val)
				if err == nil {
					usersByName[username] = User{Username: username, ProfileURL: profile.String()}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	users := make([]User, 0, len(usersByName))
	for _, user := range usersByName {
		users = append(users, user)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Username < users[j].Username })
	if len(users) == 0 {
		return nil, fmt.Errorf("parse users page: found no profile links")
	}
	return users, nil
}
