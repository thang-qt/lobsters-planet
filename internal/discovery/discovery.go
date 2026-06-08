package discovery

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"lobsters-planet/internal/config"
	"lobsters-planet/internal/lobsters"
	"lobsters-planet/internal/state"
)

type Result struct {
	FetchedUsers      int
	NewUsers          int
	KnownUsers        int
	ProfilesSelected  int
	ProfilesSucceeded int
	ProfilesFailed    int
	HomepagesFound    int
}

func Run(ctx context.Context, cfg config.Config) (Result, state.State, error) {
	store, err := state.Load(cfg.Output.StateFile)
	if err != nil {
		return Result{}, state.State{}, err
	}

	client := &http.Client{Timeout: time.Duration(cfg.Lobsters.RequestTimeoutSeconds) * time.Second}
	now := time.Now().UTC()

	users, err := lobsters.FetchUsers(ctx, client, cfg.Lobsters.UsersURL, cfg.UserAgent)
	if err != nil {
		return Result{}, state.State{}, err
	}

	result := Result{FetchedUsers: len(users)}
	result.NewUsers = store.MergeUsers(users, now)
	result.KnownUsers = len(store.Users)

	due := store.ProfilesDue(
		now,
		time.Duration(cfg.Lobsters.OldProfileRecheckDays)*24*time.Hour,
		cfg.Lobsters.MaxNewProfilesPerRun,
		cfg.Lobsters.MaxOldProfilesPerRun,
	)
	result.ProfilesSelected = len(due)

	for index, user := range due {
		if index > 0 {
			select {
			case <-ctx.Done():
				return result, store, ctx.Err()
			case <-time.After(time.Duration(cfg.Lobsters.RequestDelaySeconds) * time.Second):
			}
		}

		profile, err := lobsters.FetchProfile(ctx, client, user.ProfileURL, cfg.UserAgent)
		checkedAt := time.Now().UTC()
		if err != nil {
			store.RecordProfileFailure(user.Username, checkedAt, err)
			result.ProfilesFailed++
			fmt.Printf("profile %s failed: %v\n", user.Username, err)
			continue
		}
		store.RecordProfileSuccess(profile, checkedAt)
		result.ProfilesSucceeded++
		if profile.HomepageURL != "" {
			result.HomepagesFound++
			fmt.Printf("profile %s homepage %s\n", user.Username, profile.HomepageURL)
		} else {
			fmt.Printf("profile %s no homepage\n", user.Username)
		}
	}

	if err := state.Save(cfg.Output.StateFile, store); err != nil {
		return result, store, err
	}
	return result, store, nil
}
