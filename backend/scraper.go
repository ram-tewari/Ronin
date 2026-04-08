package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"
)

// ESPN API response types

type espnTeamEntry struct {
	Team struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	} `json:"team"`
}

type espnLeague struct {
	Teams []espnTeamEntry `json:"teams"`
}

type espnSport struct {
	Leagues []espnLeague `json:"leagues"`
}

type espnTeamsResponse struct {
	Sports    []espnSport `json:"sports"`
	Count     int         `json:"count"`
	PageIndex int         `json:"pageIndex"`
	PageCount int         `json:"pageCount"`
}

// Discovery types returned by our API

type DiscoveryTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DiscoverySport struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Teams []DiscoveryTeam `json:"teams"`
}

type DiscoveryResponse struct {
	Sports []DiscoverySport `json:"sports"`
}

// FetchNCAAMBBTeams hits the public ESPN API and returns all NCAA Men's Basketball teams.
// It handles pagination automatically since ESPN returns ~25 teams per page by default.
func FetchNCAAMBBTeams() ([]DiscoveryTeam, error) {
	client := &http.Client{Timeout: 20 * time.Second}

	var allTeams []DiscoveryTeam
	page := 1

	for {
		url := fmt.Sprintf(
			"https://site.api.espn.com/apis/site/v2/sports/basketball/mens-college-basketball/teams?limit=500&page=%d",
			page,
		)

		resp, err := client.Get(url)
		if err != nil {
			logger.Error("ESPN API request failed", 
				slog.String("error", err.Error()),
				slog.Int("page", page))
			return nil, fmt.Errorf("ESPN API request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.Error("ESPN API returned non-OK status",
				slog.Int("status_code", resp.StatusCode),
				slog.Int("page", page))
			return nil, fmt.Errorf("ESPN API returned status %d", resp.StatusCode)
		}

		var espnResp espnTeamsResponse
		if err := json.NewDecoder(resp.Body).Decode(&espnResp); err != nil {
			logger.Error("Failed to parse ESPN response",
				slog.String("error", err.Error()),
				slog.Int("page", page))
			return nil, fmt.Errorf("failed to parse ESPN response: %w", err)
		}

		for _, sport := range espnResp.Sports {
			for _, league := range sport.Leagues {
				for _, entry := range league.Teams {
					allTeams = append(allTeams, DiscoveryTeam{
						ID:   entry.Team.ID,
						Name: entry.Team.DisplayName,
					})
				}
			}
		}

		// Check if there are more pages
		if page >= espnResp.PageCount || espnResp.PageCount == 0 {
			break
		}
		page++
	}

	// Sort alphabetically by name for consistent ordering
	sort.Slice(allTeams, func(i, j int) bool {
		return allTeams[i].Name < allTeams[j].Name
	})

	logger.Info("NCAA teams fetched successfully", slog.Int("team_count", len(allTeams)))

	return allTeams, nil
}
