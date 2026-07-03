package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	UserAgent string         `yaml:"user_agent"`
	Lobsters  LobstersConfig `yaml:"lobsters"`
	Feeds     FeedsConfig    `yaml:"feeds"`
	Output    OutputConfig   `yaml:"output"`
}

type LobstersConfig struct {
	UsersURL              string `yaml:"users_url"`
	RequestTimeoutSeconds int    `yaml:"request_timeout_seconds"`
	RequestDelaySeconds   int    `yaml:"request_delay_seconds"`
	OldProfileRecheckDays int    `yaml:"old_profile_recheck_days"`
	MaxNewProfilesPerRun  int    `yaml:"max_new_profiles_per_run"`
	MaxOldProfilesPerRun  int    `yaml:"max_old_profiles_per_run"`
}

type FeedsConfig struct {
	RefreshIntervalHours    int      `yaml:"refresh_interval_hours"`
	RequestTimeoutSeconds   int      `yaml:"request_timeout_seconds"`
	RequestDelaySeconds     int      `yaml:"request_delay_seconds"`
	DiscoveryRecheckDays    int      `yaml:"discovery_recheck_days"`
	MaxSitesPerDiscoveryRun int      `yaml:"max_sites_per_discovery_run"`
	MaxFeedsPerRefreshRun   int      `yaml:"max_feeds_per_refresh_run"`
	ExcludeSiteURLs         []string `yaml:"exclude_site_urls"`
	ExcludeFeedURLs         []string `yaml:"exclude_feed_urls"`
	ExcludeSitePatterns     []string `yaml:"exclude_site_patterns"`
	CommonPaths             []string `yaml:"common_paths"`
}

type OutputConfig struct {
	StateFile        string `yaml:"state_file"`
	PublicDir        string `yaml:"public_dir"`
	UsersPerShard    int    `yaml:"users_per_shard"`
	MaxLatestEntries int    `yaml:"max_latest_entries"`
	HomepageEntries  int    `yaml:"homepage_entries"`
	PostsPerPage     int    `yaml:"posts_per_page"`
	UserEntries      int    `yaml:"user_entries"`
	SiteTitle        string `yaml:"site_title"`
	SiteURL          string `yaml:"site_url"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.UserAgent == "" {
		return Config{}, fmt.Errorf("user_agent is required")
	}
	if cfg.Lobsters.UsersURL == "" {
		return Config{}, fmt.Errorf("lobsters.users_url is required")
	}
	if cfg.Output.StateFile == "" {
		return Config{}, fmt.Errorf("output.state_file is required")
	}
	if cfg.Output.PublicDir == "" {
		return Config{}, fmt.Errorf("output.public_dir is required")
	}
	if cfg.Output.UsersPerShard <= 0 {
		cfg.Output.UsersPerShard = 500
	}
	if cfg.Output.MaxLatestEntries <= 0 {
		cfg.Output.MaxLatestEntries = 300
	}
	if cfg.Output.HomepageEntries <= 0 {
		cfg.Output.HomepageEntries = 30
	}
	if cfg.Output.PostsPerPage <= 0 {
		cfg.Output.PostsPerPage = 50
	}
	if cfg.Output.UserEntries <= 0 {
		cfg.Output.UserEntries = 20
	}
	if cfg.Output.SiteTitle == "" {
		cfg.Output.SiteTitle = "Lobsters Planet"
	}
	if cfg.Output.SiteURL == "" {
		cfg.Output.SiteURL = "http://localhost:8080"
	}
	if cfg.Lobsters.RequestTimeoutSeconds <= 0 {
		cfg.Lobsters.RequestTimeoutSeconds = 15
	}
	if cfg.Lobsters.RequestDelaySeconds <= 0 {
		cfg.Lobsters.RequestDelaySeconds = 5
	}
	if cfg.Lobsters.OldProfileRecheckDays <= 0 {
		cfg.Lobsters.OldProfileRecheckDays = 7
	}
	if cfg.Lobsters.MaxNewProfilesPerRun < 0 {
		cfg.Lobsters.MaxNewProfilesPerRun = 0
	}
	if cfg.Lobsters.MaxOldProfilesPerRun < 0 {
		cfg.Lobsters.MaxOldProfilesPerRun = 0
	}
	if cfg.Feeds.RequestTimeoutSeconds <= 0 {
		cfg.Feeds.RequestTimeoutSeconds = 15
	}
	if cfg.Feeds.RequestDelaySeconds <= 0 {
		cfg.Feeds.RequestDelaySeconds = 2
	}
	if cfg.Feeds.DiscoveryRecheckDays <= 0 {
		cfg.Feeds.DiscoveryRecheckDays = 30
	}
	if cfg.Feeds.MaxSitesPerDiscoveryRun < 0 {
		cfg.Feeds.MaxSitesPerDiscoveryRun = 0
	}
	if cfg.Feeds.MaxFeedsPerRefreshRun < 0 {
		cfg.Feeds.MaxFeedsPerRefreshRun = 0
	}
	cfg.Feeds.ExcludeSiteURLs = normalizeURLs(cfg.Feeds.ExcludeSiteURLs)
	cfg.Feeds.ExcludeFeedURLs = normalizeURLs(cfg.Feeds.ExcludeFeedURLs)
	cfg.Feeds.ExcludeSitePatterns = NormalizePatterns(cfg.Feeds.ExcludeSitePatterns)
	return cfg, nil
}

func normalizeURLs(values []string) []string {
	seen := make(map[string]bool)
	var normalized []string
	for _, value := range values {
		value = NormalizeURL(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	return normalized
}

func NormalizeURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(value, "/")
	}
	parsed.Fragment = ""
	parsed.Host = strings.ToLower(parsed.Host)
	if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}
	return parsed.String()
}

// NormalizePatterns lowercases patterns for case-insensitive matching.
func NormalizePatterns(patterns []string) []string {
	normalized := make([]string, 0, len(patterns))
	seen := make(map[string]bool)
	for _, p := range patterns {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" && !seen[p] {
			seen[p] = true
			normalized = append(normalized, p)
		}
	}
	return normalized
}

// MatchSitePatterns checks if url matches any of the heuristic patterns (case-insensitive substring match).
func MatchSitePatterns(rawURL string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	lowered := strings.ToLower(rawURL)
	for _, pattern := range patterns {
		if strings.Contains(lowered, pattern) {
			return true
		}
	}
	return false
}
