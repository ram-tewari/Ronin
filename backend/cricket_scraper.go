package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// cricapiMatch represents a match from the CricAPI response
type cricapiMatch struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	TeamInfo []struct {
		Name      string `json:"name"`
		ShortName string `json:"shortname"`
		Img       string `json:"img"`
	} `json:"teamInfo"`
	MatchType string `json:"matchType"`
}

type cricapiResponse struct {
	Data   []cricapiMatch `json:"data"`
	Status string         `json:"status"`
}

// Known international cricket teams for classification
var internationalTeams = map[string]bool{
	"India": true, "Australia": true, "England": true, "South Africa": true,
	"New Zealand": true, "Pakistan": true, "Sri Lanka": true, "Bangladesh": true,
	"West Indies": true, "Afghanistan": true, "Zimbabwe": true, "Ireland": true,
	"Netherlands": true, "Scotland": true, "Nepal": true, "Oman": true,
	"Namibia": true, "UAE": true, "USA": true, "Canada": true,
}

// Known T20 league franchise teams for classification
var t20LeagueTeams = map[string]bool{
	// IPL
	"Chennai Super Kings": true, "Mumbai Indians": true, "Royal Challengers Bengaluru": true,
	"Kolkata Knight Riders": true, "Sunrisers Hyderabad": true, "Rajasthan Royals": true,
	"Delhi Capitals": true, "Punjab Kings": true, "Gujarat Titans": true,
	"Lucknow Super Giants": true,
	// BBL
	"Melbourne Stars": true, "Sydney Sixers": true, "Perth Scorchers": true,
	"Brisbane Heat": true, "Adelaide Strikers": true, "Hobart Hurricanes": true,
	"Melbourne Renegades": true, "Sydney Thunder": true,
	// PSL
	"Karachi Kings": true, "Lahore Qalandars": true, "Islamabad United": true,
	"Peshawar Zalmi": true, "Quetta Gladiators": true, "Multan Sultans": true,
	// CPL
	"Trinbago Knight Riders": true, "Jamaica Tallawahs": true,
	"Guyana Amazon Warriors": true, "Barbados Royals": true,
	"St Kitts & Nevis Patriots": true, "Saint Lucia Kings": true,
}

// loadEnvKey reads a key from a .env file in the backend directory.
func loadEnvKey(key string) string {
	// Check OS environment first
	if val := os.Getenv(key); val != "" {
		return val
	}

	// Try reading from .env file
	f, err := os.Open(".env")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == key {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

// FetchCricketTeams fetches cricket data from CricAPI if an API key is available,
// otherwise returns a robust mocked dataset. Results are split into International
// and T20 League categories.
func FetchCricketTeams() (intlTeams []DiscoveryTeam, t20Teams []DiscoveryTeam, err error) {
	apiKey := loadEnvKey("CRICKET_API_KEY")

	if apiKey != "" {
		return fetchLiveCricketTeams(apiKey)
	}
	return getMockedCricketTeams()
}

func fetchLiveCricketTeams(apiKey string) ([]DiscoveryTeam, []DiscoveryTeam, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	url := fmt.Sprintf("https://api.cricapi.com/v1/currentMatches?apikey=%s&offset=0", apiKey)

	resp, err := client.Get(url)
	if err != nil {
		logger.Error("CricAPI request failed", slog.String("error", err.Error()))
		return nil, nil, fmt.Errorf("CricAPI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("CricAPI returned non-OK status", slog.Int("status_code", resp.StatusCode))
		return nil, nil, fmt.Errorf("CricAPI returned status %d", resp.StatusCode)
	}

	var apiResp cricapiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		logger.Error("Failed to parse CricAPI response", slog.String("error", err.Error()))
		return nil, nil, fmt.Errorf("failed to parse CricAPI response: %w", err)
	}

	intlSeen := map[string]bool{}
	t20Seen := map[string]bool{}
	var intlTeams, t20Teams []DiscoveryTeam

	for _, match := range apiResp.Data {
		for _, team := range match.TeamInfo {
			name := team.Name
			id := strings.ToLower(strings.ReplaceAll(name, " ", "_"))

			if internationalTeams[name] {
				if !intlSeen[id] {
					intlSeen[id] = true
					intlTeams = append(intlTeams, DiscoveryTeam{
						ID:   "intl_" + id,
						Name: name,
					})
				}
			} else if t20LeagueTeams[name] {
				if !t20Seen[id] {
					t20Seen[id] = true
					t20Teams = append(t20Teams, DiscoveryTeam{
						ID:   "t20_" + id,
						Name: name,
					})
				}
			} else {
				// Unknown team — classify by match type
				if match.MatchType == "t20" || match.MatchType == "t10" {
					if !t20Seen[id] {
						t20Seen[id] = true
						t20Teams = append(t20Teams, DiscoveryTeam{
							ID:   "t20_" + id,
							Name: name,
						})
					}
				} else {
					if !intlSeen[id] {
						intlSeen[id] = true
						intlTeams = append(intlTeams, DiscoveryTeam{
							ID:   "intl_" + id,
							Name: name,
						})
					}
				}
			}
		}
	}

	sort.Slice(intlTeams, func(i, j int) bool { return intlTeams[i].Name < intlTeams[j].Name })
	sort.Slice(t20Teams, func(i, j int) bool { return t20Teams[i].Name < t20Teams[j].Name })

	logger.Info("Cricket teams fetched from API",
		slog.Int("intl_teams", len(intlTeams)),
		slog.Int("t20_teams", len(t20Teams)))

	return intlTeams, t20Teams, nil
}

func getMockedCricketTeams() ([]DiscoveryTeam, []DiscoveryTeam, error) {
	logger.Info("Using mocked cricket teams (no API key found)")
	
	intlTeams := []DiscoveryTeam{
		{ID: "intl_afghanistan", Name: "Afghanistan"},
		{ID: "intl_australia", Name: "Australia"},
		{ID: "intl_bangladesh", Name: "Bangladesh"},
		{ID: "intl_england", Name: "England"},
		{ID: "intl_india", Name: "India"},
		{ID: "intl_ireland", Name: "Ireland"},
		{ID: "intl_new_zealand", Name: "New Zealand"},
		{ID: "intl_pakistan", Name: "Pakistan"},
		{ID: "intl_south_africa", Name: "South Africa"},
		{ID: "intl_sri_lanka", Name: "Sri Lanka"},
		{ID: "intl_west_indies", Name: "West Indies"},
		{ID: "intl_zimbabwe", Name: "Zimbabwe"},
	}

	t20Teams := []DiscoveryTeam{
		{ID: "t20_chennai_super_kings", Name: "Chennai Super Kings"},
		{ID: "t20_delhi_capitals", Name: "Delhi Capitals"},
		{ID: "t20_gujarat_titans", Name: "Gujarat Titans"},
		{ID: "t20_kolkata_knight_riders", Name: "Kolkata Knight Riders"},
		{ID: "t20_lucknow_super_giants", Name: "Lucknow Super Giants"},
		{ID: "t20_melbourne_stars", Name: "Melbourne Stars"},
		{ID: "t20_mumbai_indians", Name: "Mumbai Indians"},
		{ID: "t20_perth_scorchers", Name: "Perth Scorchers"},
		{ID: "t20_punjab_kings", Name: "Punjab Kings"},
		{ID: "t20_rajasthan_royals", Name: "Rajasthan Royals"},
		{ID: "t20_royal_challengers_bengaluru", Name: "Royal Challengers Bengaluru"},
		{ID: "t20_sunrisers_hyderabad", Name: "Sunrisers Hyderabad"},
		{ID: "t20_sydney_sixers", Name: "Sydney Sixers"},
		{ID: "t20_trinbago_knight_riders", Name: "Trinbago Knight Riders"},
	}

	return intlTeams, t20Teams, nil
}
