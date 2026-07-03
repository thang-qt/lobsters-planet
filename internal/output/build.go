package output

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lobsters-planet/internal/config"
	"lobsters-planet/internal/state"
)

type Result struct {
	Users      int
	Sites      int
	Entries    int
	UserShards int
	PublicDir  string
}

type PublicUser struct {
	Username         string     `json:"username"`
	ProfileURL       string     `json:"profile_url"`
	HomepageURL      string     `json:"homepage_url,omitempty"`
	HasDetailPage    bool       `json:"has_detail_page,omitempty"`
	ProfileCheckedAt *time.Time `json:"profile_checked_at,omitempty"`
	FeedCheckedAt    *time.Time `json:"feed_checked_at,omitempty"`
	UsersPageRank    int        `json:"users_page_rank,omitempty"`
	JoinedAt         *time.Time `json:"joined_at,omitempty"`
	Karma            *int       `json:"karma,omitempty"`
	StoriesSubmitted *int       `json:"stories_submitted,omitempty"`
	CommentsPosted   *int       `json:"comments_posted,omitempty"`
	About            string     `json:"about,omitempty"`
}

type PublicSite struct {
	Username         string     `json:"username"`
	ProfileURL       string     `json:"profile_url"`
	HomepageURL      string     `json:"homepage_url"`
	FeedURLs         []string   `json:"feed_urls,omitempty"`
	UsersPageRank    int        `json:"users_page_rank,omitempty"`
	JoinedAt         *time.Time `json:"joined_at,omitempty"`
	Karma            *int       `json:"karma,omitempty"`
	StoriesSubmitted *int       `json:"stories_submitted,omitempty"`
	CommentsPosted   *int       `json:"comments_posted,omitempty"`
	About            string     `json:"about,omitempty"`
}

type PublicEntry struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	URL           string     `json:"url"`
	Summary       string     `json:"summary,omitempty"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
	FeedURL       string     `json:"feed_url"`
	FeedTitle     string     `json:"feed_title,omitempty"`
	OwnerUsername string     `json:"owner_username"`
	SiteURL       string     `json:"site_url"`
}

type Stats struct {
	GeneratedAt time.Time `json:"generated_at"`
	Users       int       `json:"users"`
	Sites       int       `json:"sites"`
	Entries     int       `json:"entries"`
	UserShards  int       `json:"user_shards"`
}

type UserShardIndexEntry struct {
	Path  string `json:"path"`
	Start int    `json:"start"`
	End   int    `json:"end"`
	Count int    `json:"count"`
}

type Analytics struct {
	GeneratedAt time.Time         `json:"generated_at"`
	Overview    AnalyticsOverview `json:"overview"`
	OldestUsers []PublicUser      `json:"oldest_users"`
	TopKarma    []PublicUser      `json:"top_karma"`
	TopStories  []PublicUser      `json:"top_stories"`
	TopComments []PublicUser      `json:"top_comments"`
	ActiveFeeds []FeedAnalytics   `json:"active_feeds"`
	RecentSites []RecentSite      `json:"recent_sites"`
}

type AnalyticsOverview struct {
	Users           int `json:"users"`
	CrawledProfiles int `json:"crawled_profiles"`
	Homepages       int `json:"homepages"`
	UsersWithFeeds  int `json:"users_with_feeds"`
	Feeds           int `json:"feeds"`
	FeedsWithPosts  int `json:"feeds_with_posts"`
	Entries         int `json:"entries"`
}

type FeedAnalytics struct {
	FeedURL       string     `json:"feed_url"`
	FeedTitle     string     `json:"feed_title,omitempty"`
	OwnerUsername string     `json:"owner_username"`
	SiteURL       string     `json:"site_url"`
	Entries       int        `json:"entries"`
	LatestAt      *time.Time `json:"latest_at,omitempty"`
}

type RecentSite struct {
	Username  string     `json:"username"`
	SiteURL   string     `json:"site_url"`
	FeedTitle string     `json:"feed_title,omitempty"`
	LatestAt  *time.Time `json:"latest_at,omitempty"`
	Title     string     `json:"title,omitempty"`
	URL       string     `json:"url,omitempty"`
}

func Build(cfg config.Config) (Result, error) {
	store, err := state.Load(cfg.Output.StateFile)
	if err != nil {
		return Result{}, err
	}

	applyExclusions(&store, cfg)
	users, sites, entries := publicData(store, cfg.Output.MaxLatestEntries)
	publicDir := cfg.Output.PublicDir
	dataDir := filepath.Join(publicDir, "data")
	userDir := filepath.Join(dataDir, "users")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create public data directories: %w", err)
	}

	shards, err := writeUserShards(userDir, cfg.Output.UsersPerShard, users)
	if err != nil {
		return Result{}, err
	}

	stats := Stats{GeneratedAt: time.Now().UTC(), Users: len(users), Sites: len(sites), Entries: len(entries), UserShards: len(shards)}
	if err := writeJSON(filepath.Join(dataDir, "stats.json"), stats); err != nil {
		return Result{}, err
	}
	if err := writeJSON(filepath.Join(dataDir, "users-index.json"), shards); err != nil {
		return Result{}, err
	}
	if err := writeJSON(filepath.Join(dataDir, "sites.json"), sites); err != nil {
		return Result{}, err
	}
	if err := writeJSON(filepath.Join(publicDir, "latest.json"), entries); err != nil {
		return Result{}, err
	}
	if err := writeRSS(filepath.Join(publicDir, "feed.xml"), cfg, entries, stats.GeneratedAt); err != nil {
		return Result{}, err
	}
	analytics := buildAnalytics(stats, users, sites, entries)
	if err := writeJSON(filepath.Join(dataDir, "analytics.json"), analytics); err != nil {
		return Result{}, err
	}
	if err := writeSitePages(publicDir, cfg.Output.SiteTitle, stats, users, sites, entries, analytics, cfg.Output.UsersPerShard, cfg.Output.HomepageEntries, cfg.Output.PostsPerPage, cfg.Output.UserEntries); err != nil {
		return Result{}, err
	}

	return Result{Users: len(users), Sites: len(sites), Entries: len(entries), UserShards: len(shards), PublicDir: publicDir}, nil
}

func applyExclusions(store *state.State, cfg config.Config) {
	excludedSites := make(map[string]bool)
	for _, siteURL := range cfg.Feeds.ExcludeSiteURLs {
		excludedSites[config.NormalizeURL(siteURL)] = true
	}
	excludedFeeds := make(map[string]bool)
	for _, feedURL := range cfg.Feeds.ExcludeFeedURLs {
		excludedFeeds[config.NormalizeURL(feedURL)] = true
	}
	patterns := cfg.Feeds.ExcludeSitePatterns

	for username, user := range store.Users {
		if excludedSites[config.NormalizeURL(user.HomepageURL)] || config.MatchSitePatterns(user.HomepageURL, patterns) {
			for _, feedURL := range user.FeedURLs {
				excludedFeeds[config.NormalizeURL(feedURL)] = true
			}
			user.HomepageURL = ""
			user.FeedURLs = nil
			store.Users[username] = user
			continue
		}
		if len(user.FeedURLs) > 0 {
			filtered := user.FeedURLs[:0]
			for _, feedURL := range user.FeedURLs {
				if !excludedFeeds[config.NormalizeURL(feedURL)] {
					filtered = append(filtered, feedURL)
				}
			}
			user.FeedURLs = filtered
			store.Users[username] = user
		}
	}
	for feedURL, feed := range store.Feeds {
		if excludedFeeds[config.NormalizeURL(feedURL)] || excludedSites[config.NormalizeURL(feed.SiteURL)] || config.MatchSitePatterns(feed.SiteURL, patterns) {
			delete(store.Feeds, feedURL)
		}
	}
	for entryID, entry := range store.Entries {
		if excludedFeeds[config.NormalizeURL(entry.FeedURL)] || excludedSites[config.NormalizeURL(entry.SiteURL)] || config.MatchSitePatterns(entry.SiteURL, patterns) {
			delete(store.Entries, entryID)
		}
	}
}

func publicData(store state.State, maxEntries int) ([]PublicUser, []PublicSite, []PublicEntry) {
	users := make([]PublicUser, 0, len(store.Users))
	sites := make([]PublicSite, 0)
	for _, user := range store.Users {
		publicUser := PublicUser{
			Username:         user.Username,
			ProfileURL:       user.ProfileURL,
			HomepageURL:      user.HomepageURL,
			HasDetailPage:    user.JoinedAt != nil || user.HomepageURL != "" || user.About != "",
			ProfileCheckedAt: user.ProfileCheckedAt,
			FeedCheckedAt:    latestFeedCheck(user, store),
			UsersPageRank:    user.UsersPageRank,
			JoinedAt:         user.JoinedAt,
			Karma:            user.Karma,
			StoriesSubmitted: user.StoriesSubmitted,
			CommentsPosted:   user.CommentsPosted,
			About:            user.About,
		}
		users = append(users, publicUser)
		if user.HomepageURL != "" {
			sites = append(sites, PublicSite{
				Username:         user.Username,
				ProfileURL:       user.ProfileURL,
				HomepageURL:      user.HomepageURL,
				FeedURLs:         user.FeedURLs,
				UsersPageRank:    user.UsersPageRank,
				JoinedAt:         user.JoinedAt,
				Karma:            user.Karma,
				StoriesSubmitted: user.StoriesSubmitted,
				CommentsPosted:   user.CommentsPosted,
				About:            user.About,
			})
		}
	}
	sort.Slice(users, func(i, j int) bool { return compareUsers(users[i], users[j]) < 0 })
	sort.Slice(sites, func(i, j int) bool { return compareSites(sites[i], sites[j]) < 0 })
	entries := publicEntries(store, maxEntries)
	return users, sites, entries
}

func latestFeedCheck(user state.DiscoveredUser, store state.State) *time.Time {
	var latest *time.Time
	for _, feedURL := range user.FeedURLs {
		checkedAt := store.Feeds[feedURL].CheckedAt
		if checkedAt != nil && (latest == nil || checkedAt.After(*latest)) {
			latest = checkedAt
		}
	}
	return latest
}

func publicEntries(store state.State, maxEntries int) []PublicEntry {
	entries := make([]PublicEntry, 0, len(store.Entries))
	for _, entry := range store.Entries {
		entries = append(entries, PublicEntry{
			ID: entry.ID, Title: entry.Title, URL: entry.URL, Summary: entry.Summary,
			PublishedAt: entry.PublishedAt, UpdatedAt: entry.UpdatedAt, FeedURL: entry.FeedURL,
			FeedTitle: entry.FeedTitle, OwnerUsername: entry.OwnerUsername, SiteURL: entry.SiteURL,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entryTime(entries[i]).After(entryTime(entries[j])) })
	if maxEntries > 0 && len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}
	return entries
}

func entryTime(entry PublicEntry) time.Time {
	if entry.PublishedAt != nil {
		return *entry.PublishedAt
	}
	if entry.UpdatedAt != nil {
		return *entry.UpdatedAt
	}
	return time.Time{}
}

func compareUsers(a, b PublicUser) int {
	if a.UsersPageRank != 0 && b.UsersPageRank != 0 && a.UsersPageRank != b.UsersPageRank {
		if a.UsersPageRank < b.UsersPageRank {
			return -1
		}
		return 1
	}
	return strings.Compare(a.Username, b.Username)
}

func compareSites(a, b PublicSite) int {
	if a.UsersPageRank != 0 && b.UsersPageRank != 0 && a.UsersPageRank != b.UsersPageRank {
		if a.UsersPageRank < b.UsersPageRank {
			return -1
		}
		return 1
	}
	return strings.Compare(a.Username, b.Username)
}

func writeUserShards(userDir string, size int, users []PublicUser) ([]UserShardIndexEntry, error) {
	var shards []UserShardIndexEntry
	for start := 0; start < len(users); start += size {
		end := min(start+size, len(users))
		path := fmt.Sprintf("users/%05d-%05d.json", start+1, end)
		if err := writeJSON(filepath.Join(filepath.Dir(userDir), path), users[start:end]); err != nil {
			return nil, err
		}
		shards = append(shards, UserShardIndexEntry{Path: path, Start: start + 1, End: end, Count: end - start})
	}
	return shards, nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	return writeFile(path, data)
}

func buildAnalytics(stats Stats, users []PublicUser, sites []PublicSite, entries []PublicEntry) Analytics {
	analytics := Analytics{GeneratedAt: stats.GeneratedAt}
	analytics.Overview.Users = len(users)
	analytics.Overview.Homepages = len(sites)
	analytics.Overview.Entries = len(entries)

	for _, user := range users {
		if user.ProfileCheckedAt != nil {
			analytics.Overview.CrawledProfiles++
		}
		if user.FeedCheckedAt != nil {
			analytics.Overview.UsersWithFeeds++
		}
	}

	feeds := make(map[string]FeedAnalytics)
	latestByUser := make(map[string]RecentSite)
	for _, entry := range entries {
		feed := feeds[entry.FeedURL]
		feed.FeedURL = entry.FeedURL
		feed.FeedTitle = entry.FeedTitle
		feed.OwnerUsername = entry.OwnerUsername
		feed.SiteURL = entry.SiteURL
		feed.Entries++
		entryAt := entryTime(entry)
		if !entryAt.IsZero() && (feed.LatestAt == nil || entryAt.After(*feed.LatestAt)) {
			feed.LatestAt = &entryAt
		}
		feeds[entry.FeedURL] = feed

		recent := latestByUser[entry.OwnerUsername]
		if recent.LatestAt == nil || (!entryAt.IsZero() && entryAt.After(*recent.LatestAt)) {
			recent = RecentSite{Username: entry.OwnerUsername, SiteURL: entry.SiteURL, FeedTitle: entry.FeedTitle, LatestAt: nil, Title: entry.Title, URL: entry.URL}
			if !entryAt.IsZero() {
				recent.LatestAt = &entryAt
			}
			latestByUser[entry.OwnerUsername] = recent
		}
	}
	analytics.Overview.Feeds = len(feeds)
	for _, feed := range feeds {
		analytics.ActiveFeeds = append(analytics.ActiveFeeds, feed)
		if feed.Entries > 0 {
			analytics.Overview.FeedsWithPosts++
		}
	}
	for _, recent := range latestByUser {
		analytics.RecentSites = append(analytics.RecentSites, recent)
	}

	oldest := filterUsers(users, func(user PublicUser) bool { return user.JoinedAt != nil })
	sort.Slice(oldest, func(i, j int) bool { return oldest[i].JoinedAt.Before(*oldest[j].JoinedAt) })
	analytics.OldestUsers = limitUsers(oldest, 25)

	karma := filterUsers(users, func(user PublicUser) bool { return user.Karma != nil })
	sort.Slice(karma, func(i, j int) bool { return *karma[i].Karma > *karma[j].Karma })
	analytics.TopKarma = limitUsers(karma, 25)

	stories := filterUsers(users, func(user PublicUser) bool { return user.StoriesSubmitted != nil })
	sort.Slice(stories, func(i, j int) bool { return *stories[i].StoriesSubmitted > *stories[j].StoriesSubmitted })
	analytics.TopStories = limitUsers(stories, 25)

	comments := filterUsers(users, func(user PublicUser) bool { return user.CommentsPosted != nil })
	sort.Slice(comments, func(i, j int) bool { return *comments[i].CommentsPosted > *comments[j].CommentsPosted })
	analytics.TopComments = limitUsers(comments, 25)

	sort.Slice(analytics.ActiveFeeds, func(i, j int) bool {
		if analytics.ActiveFeeds[i].Entries != analytics.ActiveFeeds[j].Entries {
			return analytics.ActiveFeeds[i].Entries > analytics.ActiveFeeds[j].Entries
		}
		return analytics.ActiveFeeds[i].OwnerUsername < analytics.ActiveFeeds[j].OwnerUsername
	})
	analytics.ActiveFeeds = limitFeeds(analytics.ActiveFeeds, 25)

	sort.Slice(analytics.RecentSites, func(i, j int) bool {
		if analytics.RecentSites[i].LatestAt == nil {
			return false
		}
		if analytics.RecentSites[j].LatestAt == nil {
			return true
		}
		return analytics.RecentSites[i].LatestAt.After(*analytics.RecentSites[j].LatestAt)
	})
	analytics.RecentSites = limitRecentSites(analytics.RecentSites, 25)

	return analytics
}

func filterUsers(users []PublicUser, keep func(PublicUser) bool) []PublicUser {
	filtered := make([]PublicUser, 0)
	for _, user := range users {
		if keep(user) {
			filtered = append(filtered, user)
		}
	}
	return filtered
}

func limitUsers(users []PublicUser, max int) []PublicUser {
	if len(users) <= max {
		return users
	}
	return users[:max]
}

func limitFeeds(feeds []FeedAnalytics, max int) []FeedAnalytics {
	if len(feeds) <= max {
		return feeds
	}
	return feeds[:max]
}

func limitRecentSites(sites []RecentSite, max int) []RecentSite {
	if len(sites) <= max {
		return sites
	}
	return sites[:max]
}

func writeSitePages(publicDir, title string, stats Stats, users []PublicUser, sites []PublicSite, entries []PublicEntry, analytics Analytics, usersPerPage, homepageEntries, postsPerPage, userEntries int) error {
	entriesByUser := make(map[string][]PublicEntry)
	for _, entry := range entries {
		entriesByUser[entry.OwnerUsername] = append(entriesByUser[entry.OwnerUsername], entry)
	}

	if err := renderPage(filepath.Join(publicDir, "index.html"), homeTemplate, map[string]any{"Title": title, "Stats": stats, "Entries": entries[:min(homepageEntries, len(entries))], "Users": users[:min(18, len(users))], "Sites": sites}); err != nil {
		return err
	}
	if err := writePostPages(publicDir, title, stats, entries, postsPerPage); err != nil {
		return err
	}
	if err := renderPage(filepath.Join(publicDir, "explore", "index.html"), exploreTemplate, map[string]any{"Title": "Explore — " + title}); err != nil {
		return err
	}
	if err := renderPage(filepath.Join(publicDir, "analytics", "index.html"), analyticsTemplate, map[string]any{"Title": title, "Analytics": analytics}); err != nil {
		return err
	}
	if err := renderPage(filepath.Join(publicDir, "users", "index.html"), usersTemplate, map[string]any{"Title": title, "Stats": stats, "Users": users[:min(usersPerPage, len(users))], "Page": 1, "Pages": (len(users) + usersPerPage - 1) / usersPerPage}); err != nil {
		return err
	}
	pages := (len(users) + usersPerPage - 1) / usersPerPage
	for page := 2; page <= pages; page++ {
		start, end := (page-1)*usersPerPage, min(page*usersPerPage, len(users))
		if err := renderPage(filepath.Join(publicDir, "users", fmt.Sprintf("page-%d.html", page)), usersTemplate, map[string]any{"Title": title, "Stats": stats, "Users": users[start:end], "Page": page, "Pages": pages}); err != nil {
			return err
		}
	}
	for _, user := range users {
		if !user.HasDetailPage {
			continue
		}
		userPosts := entriesByUser[user.Username]
		if len(userPosts) > userEntries {
			userPosts = userPosts[:userEntries]
		}
		if err := renderPage(filepath.Join(publicDir, "users", user.Username, "index.html"), userTemplate, map[string]any{"Title": title, "User": user, "Entries": userPosts}); err != nil {
			return err
		}
	}
	return nil
}

func writePostPages(publicDir, title string, stats Stats, entries []PublicEntry, postsPerPage int) error {
	pages := (len(entries) + postsPerPage - 1) / postsPerPage
	if pages == 0 {
		pages = 1
	}
	for page := 1; page <= pages; page++ {
		start, end := (page-1)*postsPerPage, min(page*postsPerPage, len(entries))
		path := filepath.Join(publicDir, "posts", "index.html")
		if page > 1 {
			path = filepath.Join(publicDir, "posts", fmt.Sprintf("page-%d.html", page))
		}
		if err := renderPage(path, postsTemplate, map[string]any{"Title": title, "Stats": stats, "Entries": entries[start:end], "Page": page, "Pages": pages}); err != nil {
			return err
		}
	}
	return nil
}

func renderPage(path, pageTemplate string, data any) error {
	functions := template.FuncMap{
		"date": func(value *time.Time) string {
			if value == nil {
				return ""
			}
			return value.Format("2 Jan 2006")
		},
		"datetime": func(value time.Time) string {
			if value.IsZero() {
				return ""
			}
			return value.Format("2 Jan 2006, 15:04 UTC")
		},
		"entryDate": func(entry PublicEntry) string {
			value := entryTime(entry)
			if value.IsZero() {
				return ""
			}
			return value.Format("2 Jan 2006")
		},
		"userPath": func(username string) string { return "/users/" + username + "/" },
		"prevPath": func(page int) string {
			if page <= 2 {
				return "/users/"
			}
			return fmt.Sprintf("/users/page-%d.html", page-1)
		},
		"nextPath": func(page int) string { return fmt.Sprintf("/users/page-%d.html", page+1) },
		"prevPostPath": func(page int) string {
			if page <= 2 {
				return "/posts/"
			}
			return fmt.Sprintf("/posts/page-%d.html", page-1)
		},
		"nextPostPath": func(page int) string { return fmt.Sprintf("/posts/page-%d.html", page+1) },
	}
	tmpl, err := template.New("page").Funcs(functions).Parse(pageTemplate)
	if err != nil {
		return fmt.Errorf("parse template for %s: %w", path, err)
	}
	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return fmt.Errorf("render %s: %w", path, err)
	}
	return writeFile(path, []byte(builder.String()))
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", temporary, err)
	}
	if err := os.Rename(temporary, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
