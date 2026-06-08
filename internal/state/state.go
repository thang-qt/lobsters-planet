package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"lobsters-planet/internal/lobsters"
)

type State struct {
	Version      int                       `json:"version"`
	UsersChecked time.Time                 `json:"users_checked_at"`
	Users        map[string]DiscoveredUser `json:"users"`
}

type DiscoveredUser struct {
	Username         string     `json:"username"`
	ProfileURL       string     `json:"profile_url"`
	DiscoveredAt     time.Time  `json:"discovered_at"`
	LastSeenAt       time.Time  `json:"last_seen_at"`
	ProfileCheckedAt *time.Time `json:"profile_checked_at,omitempty"`
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
		existing.LastSeenAt = checkedAt
		s.Users[user.Username] = existing
	}
	s.UsersChecked = checkedAt
	return newUsers
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
