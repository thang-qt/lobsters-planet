package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"lobsters-planet/internal/lobsters"
)

type State struct {
	Version      int                       `json:"version"`
	UsersChecked time.Time                 `json:"users_checked_at"`
	Users        map[string]DiscoveredUser `json:"users"`
}

type DiscoveredUser struct {
	Username                  string     `json:"username"`
	ProfileURL                string     `json:"profile_url"`
	HomepageURL               string     `json:"homepage_url,omitempty"`
	UsersPageRank             int        `json:"users_page_rank,omitempty"`
	JoinedAt                  *time.Time `json:"joined_at,omitempty"`
	Karma                     *int       `json:"karma,omitempty"`
	StoriesSubmitted          *int       `json:"stories_submitted,omitempty"`
	CommentsPosted            *int       `json:"comments_posted,omitempty"`
	About                     string     `json:"about,omitempty"`
	FeedURLs                  []string   `json:"feed_urls,omitempty"`
	FeedDiscoveryCheckedAt    *time.Time `json:"feed_discovery_checked_at,omitempty"`
	FeedDiscoveryLastError    string     `json:"feed_discovery_last_error,omitempty"`
	FeedDiscoveryFailureCount int        `json:"feed_discovery_failure_count,omitempty"`
	DiscoveredAt              time.Time  `json:"discovered_at"`
	LastSeenAt                time.Time  `json:"last_seen_at"`
	ProfileCheckedAt          *time.Time `json:"profile_checked_at,omitempty"`
	ProfileLastError          string     `json:"profile_last_error,omitempty"`
	ProfileFailureCount       int        `json:"profile_failure_count,omitempty"`
}

func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{Version: 1, Users: make(map[string]DiscoveredUser)}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read state: %w", err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse state: %w", err)
	}
	if state.Users == nil {
		state.Users = make(map[string]DiscoveredUser)
	}
	return state, nil
}

func (s *State) MergeUsers(users []lobsters.User, checkedAt time.Time) int {
	newUsers := 0
	for _, user := range users {
		existing, found := s.Users[user.Username]
		if !found {
			newUsers++
			existing.DiscoveredAt = checkedAt
		}
		existing.Username = user.Username
		existing.ProfileURL = user.ProfileURL
		existing.UsersPageRank = user.UsersPageRank
		existing.LastSeenAt = checkedAt
		s.Users[user.Username] = existing
	}
	s.UsersChecked = checkedAt
	return newUsers
}

// ProfilesDue selects never-checked profiles first, then stale profiles.
//
// Never-checked profiles are ordered by Lobste.rs users-page rank. That rank is
// invitation-tree order, which starts at the oldest/root accounts and gives us a
// useful seed set. Stale profiles are ordered by a lightweight activity heuristic
// so scarce refresh slots favor users with more visible participation.
func (s State) ProfilesDue(now time.Time, recheckAfter time.Duration, maxNew, maxOld int) []DiscoveredUser {
	var fresh, stale []DiscoveredUser
	for _, user := range s.Users {
		if user.ProfileCheckedAt == nil {
			fresh = append(fresh, user)
		} else if now.Sub(*user.ProfileCheckedAt) >= recheckAfter {
			stale = append(stale, user)
		}
	}
	sort.Slice(fresh, func(i, j int) bool { return compareUnchecked(fresh[i], fresh[j]) < 0 })
	sort.Slice(stale, func(i, j int) bool { return priorityScore(stale[i], now) > priorityScore(stale[j], now) })
	fresh = limit(fresh, maxNew)
	stale = limit(stale, maxOld)
	return append(fresh, stale...)
}

func compareUnchecked(a, b DiscoveredUser) int {
	if a.UsersPageRank != 0 && b.UsersPageRank != 0 && a.UsersPageRank != b.UsersPageRank {
		if a.UsersPageRank < b.UsersPageRank {
			return -1
		}
		return 1
	}
	if !a.DiscoveredAt.Equal(b.DiscoveredAt) {
		if a.DiscoveredAt.Before(b.DiscoveredAt) {
			return -1
		}
		return 1
	}
	if a.Username < b.Username {
		return -1
	}
	if a.Username > b.Username {
		return 1
	}
	return 0
}

func priorityScore(user DiscoveredUser, now time.Time) int {
	score := 0
	if user.Karma != nil {
		score += *user.Karma
	}
	if user.StoriesSubmitted != nil {
		score += *user.StoriesSubmitted * 20
	}
	if user.CommentsPosted != nil {
		score += *user.CommentsPosted * 2
	}
	if user.JoinedAt != nil {
		ageDays := int(now.Sub(*user.JoinedAt).Hours() / 24)
		score += min(ageDays, 3650) / 10
	}
	if user.ProfileCheckedAt != nil {
		staleDays := int(now.Sub(*user.ProfileCheckedAt).Hours() / 24)
		score += staleDays * 5
	}
	if user.HomepageURL != "" {
		score += 100
	}
	return score
}

func limit(users []DiscoveredUser, max int) []DiscoveredUser {
	if max == 0 || len(users) <= max {
		return users
	}
	return users[:max]
}

func (s *State) RecordProfileSuccess(profile lobsters.Profile, checkedAt time.Time) {
	user := s.Users[profile.Username]
	user.HomepageURL = profile.HomepageURL
	user.JoinedAt = profile.JoinedAt
	user.Karma = profile.Karma
	user.StoriesSubmitted = profile.StoriesSubmitted
	user.CommentsPosted = profile.CommentsPosted
	user.About = profile.About
	user.ProfileCheckedAt = &checkedAt
	user.ProfileLastError = ""
	user.ProfileFailureCount = 0
	s.Users[profile.Username] = user
}

func (s *State) RecordProfileFailure(username string, checkedAt time.Time, profileError error) {
	user := s.Users[username]
	user.ProfileCheckedAt = &checkedAt
	user.ProfileLastError = profileError.Error()
	user.ProfileFailureCount++
	s.Users[username] = user
}

func (s State) SitesDueForFeedDiscovery(now time.Time, recheckAfter time.Duration, maxSites int) []DiscoveredUser {
	var due []DiscoveredUser
	for _, user := range s.Users {
		if user.HomepageURL == "" {
			continue
		}
		if user.FeedDiscoveryCheckedAt == nil || now.Sub(*user.FeedDiscoveryCheckedAt) >= recheckAfter {
			due = append(due, user)
		}
	}
	sort.Slice(due, func(i, j int) bool { return priorityScore(due[i], now) > priorityScore(due[j], now) })
	return limit(due, maxSites)
}

func (s *State) RecordFeedDiscoverySuccess(username string, feedURLs []string, checkedAt time.Time) {
	user := s.Users[username]
	user.FeedURLs = feedURLs
	user.FeedDiscoveryCheckedAt = &checkedAt
	user.FeedDiscoveryLastError = ""
	user.FeedDiscoveryFailureCount = 0
	s.Users[username] = user
}

func (s *State) RecordFeedDiscoveryFailure(username string, checkedAt time.Time, discoveryError error) {
	user := s.Users[username]
	user.FeedDiscoveryCheckedAt = &checkedAt
	user.FeedDiscoveryLastError = discoveryError.Error()
	user.FeedDiscoveryFailureCount++
	s.Users[username] = user
}

func Save(path string, state State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return fmt.Errorf("write temporary state: %w", err)
	}
	if err := os.Rename(temporary, path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}
