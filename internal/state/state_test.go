package state

import (
	"testing"
	"time"
)

func TestProfilesDuePrioritizesUncheckedByUsersPageRank(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	store := State{Users: map[string]DiscoveredUser{
		"third":  {Username: "third", UsersPageRank: 3, DiscoveredAt: now},
		"first":  {Username: "first", UsersPageRank: 1, DiscoveredAt: now},
		"second": {Username: "second", UsersPageRank: 2, DiscoveredAt: now},
	}}

	due := store.ProfilesDue(now, 7*24*time.Hour, 3, 0)
	if len(due) != 3 || due[0].Username != "first" || due[1].Username != "second" || due[2].Username != "third" {
		t.Fatalf("unexpected due profiles: %#v", due)
	}
}

func TestProfilesDueIncludesStaleOldProfiles(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	old := now.Add(-8 * 24 * time.Hour)
	recent := now.Add(-time.Hour)
	karma := 100
	store := State{Users: map[string]DiscoveredUser{
		"new":    {Username: "new", UsersPageRank: 1, DiscoveredAt: now.Add(-time.Hour)},
		"old":    {Username: "old", ProfileCheckedAt: &old, Karma: &karma},
		"recent": {Username: "recent", ProfileCheckedAt: &recent},
	}}

	due := store.ProfilesDue(now, 7*24*time.Hour, 1, 1)
	if len(due) != 2 || due[0].Username != "new" || due[1].Username != "old" {
		t.Fatalf("unexpected due profiles: %#v", due)
	}
}
