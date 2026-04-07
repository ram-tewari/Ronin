package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LiveScoreCache holds thread-safe live score data
var (
	liveScoreCache   = make(map[string]string) // teamID -> live status string
	liveScoreCacheMu sync.RWMutex
)

// ESPN Scoreboard API response types
type espnScoreboardResponse struct {
	Events []espnEvent `json:"events"`
}

type espnEvent struct {
	Name         string            `json:"name"`
	Status       espnStatus        `json:"status"`
	Competitions []espnCompetition `json:"competitions"`
}

type espnStatus struct {
	Type struct {
		State       string `json:"state"`
		Completed   bool   `json:"completed"`
		Description string `json:"description"`
	} `json:"type"`
	DisplayClock string `json:"displayClock"`
	Period       int    `json:"period"`
}

type espnCompetition struct {
	Competitors []espnCompetitor `json:"competitors"`
}

type espnCompetitor struct {
	ID       string `json:"id"`
	Team     struct {
		DisplayName string `json:"displayName"`
	} `json:"team"`
	Score    string `json:"score"`
	HomeAway string `json:"homeAway"`
}

// CricAPI currentMatches response types
type cricapiCurrentMatchesResponse struct {
	Data []cricapiCurrentMatch `json:"data"`
}

type cricapiCurrentMatch struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	TeamInfo []struct {
		Name      string `json:"name"`
		ShortName string `json:"shortname"`
	} `json:"teamInfo"`
}

// StartLiveScorePoller launches a background goroutine that polls live scores every 60 seconds
func StartLiveScorePoller() {
	ticker := time.NewTicker(60 * time.Second)
	
	// Run immediately on startup
	go pollLiveScores()
	
	go func() {
		for range ticker.C {
			pollLiveScores()
		}
	}()
	
	log.Println("Live score poller started (60s interval)")
}

// pollLiveScores fetches live scores from ESPN and CricAPI and updates the cache
func pollLiveScores() {
	configMu.RLock()
	selectedTeams := make([]string, len(appConfig.SelectedTeams))
	copy(selectedTeams, appConfig.SelectedTeams)
	configMu.RUnlock()

	if len(selectedTeams) == 0 {
		return
	}

	// Separate teams by sport
	var ncaaTeams, cricketTeams []string
	for _, teamID := range selectedTeams {
		if strings.HasPrefix(teamID, "intl_") || strings.HasPrefix(teamID, "t20_") {
			cricketTeams = append(cricketTeams, teamID)
		} else {
			ncaaTeams = append(ncaaTeams, teamID)
		}
	}

	// Fetch scores concurrently
	var wg sync.WaitGroup
	
	if len(ncaaTeams) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fetchESPNScores(ncaaTeams)
		}()
	}
	
	if len(cricketTeams) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fetchCricketScores(cricketTeams)
		}()
	}
	
	wg.Wait()
}

// fetchESPNScores polls the ESPN scoreboard API for NCAA basketball games
func fetchESPNScores(teamIDs []string) {
	client := &http.Client{Timeout: 15 * time.Second}
	
	url := "https://site.api.espn.com/apis/site/v2/sports/basketball/mens-college-basketball/scoreboard"
	
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("ESPN scoreboard fetch failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("ESPN scoreboard returned status %d", resp.StatusCode)
		return
	}

	var scoreboard espnScoreboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&scoreboard); err != nil {
		log.Printf("Failed to parse ESPN scoreboard: %v", err)
		return
	}

	// Build a map of team IDs to their live status
	teamStatusMap := make(map[string]string)
	
	for _, event := range scoreboard.Events {
		for _, comp := range event.Competitions {
			if len(comp.Competitors) < 2 {
				continue
			}

			// Extract both teams
			var home, away espnCompetitor
			for _, competitor := range comp.Competitors {
				if competitor.HomeAway == "home" {
					home = competitor
				} else {
					away = competitor
				}
			}

			// Format the status string
			var statusStr string
			if event.Status.Type.Completed {
				statusStr = fmt.Sprintf("Final: %s %s - %s %s",
					home.Team.DisplayName, home.Score,
					away.Team.DisplayName, away.Score)
			} else if event.Status.Type.State == "in" {
				period := "1st Half"
				if event.Status.Period == 2 {
					period = "2nd Half"
				}
				statusStr = fmt.Sprintf("%s %s - %s %s (%s %s)",
					home.Team.DisplayName, home.Score,
					away.Team.DisplayName, away.Score,
					period, event.Status.DisplayClock)
			} else if event.Status.Type.State == "pre" {
				statusStr = fmt.Sprintf("Upcoming: %s vs %s (%s)",
					home.Team.DisplayName, away.Team.DisplayName,
					event.Status.Type.Description)
			}

			// Map both team IDs to this status
			if statusStr != "" {
				teamStatusMap[home.ID] = statusStr
				teamStatusMap[away.ID] = statusStr
			}
		}
	}

	// Update cache for tracked teams
	liveScoreCacheMu.Lock()
	for _, teamID := range teamIDs {
		if status, found := teamStatusMap[teamID]; found {
			liveScoreCache[teamID] = status
		} else {
			liveScoreCache[teamID] = "No game today"
		}
	}
	liveScoreCacheMu.Unlock()
}

// fetchCricketScores polls the CricAPI for live cricket matches
func fetchCricketScores(teamIDs []string) {
	apiKey := loadEnvKey("CRICKET_API_KEY")
	if apiKey == "" {
		// Set fallback status for all cricket teams
		liveScoreCacheMu.Lock()
		for _, teamID := range teamIDs {
			liveScoreCache[teamID] = "No game today"
		}
		liveScoreCacheMu.Unlock()
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	
	url := fmt.Sprintf("https://api.cricapi.com/v1/currentMatches?apikey=%s&offset=0", apiKey)
	
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("CricAPI fetch failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("CricAPI returned status %d", resp.StatusCode)
		return
	}

	var cricResp cricapiCurrentMatchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&cricResp); err != nil {
		log.Printf("Failed to parse CricAPI response: %v", err)
		return
	}

	// Build a map of team names to their live status
	teamStatusMap := make(map[string]string)
	
	for _, match := range cricResp.Data {
		// The status field contains the live score string
		status := match.Status
		if status == "" {
			status = match.Name
		}

		// Map all teams in this match to the status
		for _, team := range match.TeamInfo {
			teamName := team.Name
			teamStatusMap[teamName] = status
		}
	}

	// Update cache for tracked teams
	liveScoreCacheMu.Lock()
	for _, teamID := range teamIDs {
		// Extract team name from ID (remove "intl_" or "t20_" prefix)
		teamName := strings.TrimPrefix(teamID, "intl_")
		teamName = strings.TrimPrefix(teamName, "t20_")
		teamName = strings.ReplaceAll(teamName, "_", " ")
		teamName = strings.Title(strings.ToLower(teamName))

		// Try to find a match for this team
		found := false
		for apiTeamName, status := range teamStatusMap {
			if strings.EqualFold(apiTeamName, teamName) || 
			   strings.Contains(strings.ToLower(apiTeamName), strings.ToLower(teamName)) {
				liveScoreCache[teamID] = status
				found = true
				break
			}
		}
		
		if !found {
			liveScoreCache[teamID] = "No game today"
		}
	}
	liveScoreCacheMu.Unlock()
}

// GetLiveScore retrieves the cached live score for a team ID
func GetLiveScore(teamID string) string {
	liveScoreCacheMu.RLock()
	defer liveScoreCacheMu.RUnlock()
	
	if status, ok := liveScoreCache[teamID]; ok {
		return status
	}
	return "Loading..."
}

// IsGameActive checks if a status string indicates an active game
func IsGameActive(status string) bool {
	status = strings.ToLower(status)
	return !strings.Contains(status, "final") && 
	       !strings.Contains(status, "no game") &&
	       !strings.Contains(status, "loading") &&
	       !strings.Contains(status, "upcoming")
}
