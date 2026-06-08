package lobsters

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type User struct {
	Username      string `json:"username"`
	ProfileURL    string `json:"profile_url"`
	UsersPageRank int    `json:"users_page_rank"`
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

	users := collectUsersFromTrees(doc, base)
	if len(users) == 0 {
		users = collectUsersFromDocument(doc, base)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("parse users page: found no profile links")
	}
	return users, nil
}

func collectUsersFromTrees(doc *html.Node, base *url.URL) []User {
	seen := make(map[string]bool)
	var users []User
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "ul" && hasClass(n, "user_tree") {
			collectUsers(n, base, seen, &users)
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return users
}

func collectUsersFromDocument(doc *html.Node, base *url.URL) []User {
	seen := make(map[string]bool)
	var users []User
	collectUsers(doc, base, seen, &users)
	return users
}

func collectUsers(n *html.Node, base *url.URL, seen map[string]bool, users *[]User) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key != "href" || !strings.HasPrefix(attr.Val, "/~") {
				continue
			}
			username := strings.TrimPrefix(attr.Val, "/~")
			if username == "" || strings.Contains(username, "/") || seen[username] {
				continue
			}
			profile, err := base.Parse(attr.Val)
			if err == nil {
				seen[username] = true
				*users = append(*users, User{Username: username, ProfileURL: profile.String(), UsersPageRank: len(*users) + 1})
			}
		}
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		collectUsers(child, base, seen, users)
	}
}

func hasClass(n *html.Node, className string) bool {
	for _, attr := range n.Attr {
		if attr.Key != "class" {
			continue
		}
		for _, class := range strings.Fields(attr.Val) {
			if class == className {
				return true
			}
		}
	}
	return false
}
