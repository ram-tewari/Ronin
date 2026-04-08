package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
)

// Alert represents a sports alert in the Oracle report
type Alert struct {
	TeamID   string `json:"teamId"`
	Team     string `json:"team"`
	Event    string `json:"event"`
	Link     string `json:"link"`
	Priority string `json:"priority"` // high, medium, low
}

// StatusResponse represents the /status Oracle report
type StatusResponse struct {
	Mood    string  `json:"mood"` // idle, alert, hyped, exhausted
	Message string  `json:"message"`
	Alerts  []Alert `json:"alerts"`
}

// AppConfig holds the user's selected team IDs
type AppConfig struct {
	SelectedTeams []string `json:"selectedTeams"`
}

const configFileName = "ronin_config.json"

var (
	appConfig  AppConfig
	configMu   sync.RWMutex
	teamCache  map[string]string // id -> displayName
	teamCacheMu sync.RWMutex
)

func main() {
	// Initialize structured logger
	if err := InitLogger(); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	loadConfig()

	// Start the live score poller in the background
	go StartLiveScorePoller()

	mux := http.NewServeMux()
	mux.HandleFunc("/status", statusHandler)
	mux.HandleFunc("/discovery", discoveryHandler)
	mux.HandleFunc("/config", configHandler)
	mux.HandleFunc("/query", queryHandler)
	mux.HandleFunc("/log", logHandler)

	handler := corsMiddleware(mux)

	logger.Info("Starting Ronin brain", slog.String("address", "http://localhost:8080"))
	if err := http.ListenAndServe(":8080", handler); err != nil {
		logger.Error("Server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// statusHandler generates the Oracle report filtered by selected teams
func statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	configMu.RLock()
	selected := make([]string, len(appConfig.SelectedTeams))
	copy(selected, appConfig.SelectedTeams)
	configMu.RUnlock()

	if len(selected) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(StatusResponse{
			Mood:    "idle",
			Message: "No teams tracked. Open Settings to select teams.",
			Alerts:  []Alert{},
		})
		return
	}

	// Build alerts for each selected team with live scores
	var alerts []Alert
	hasActiveGame := false
	
	teamCacheMu.RLock()
	for i, id := range selected {
		name, ok := teamCache[id]
		if !ok {
			name = "Team " + id
		}

		// Get live score from cache
		liveStatus := GetLiveScore(id)
		
		// Check if this team has an active game
		if IsGameActive(liveStatus) {
			hasActiveGame = true
		}

		priority := "medium"
		if i == 0 {
			priority = "high"
		}

		// Generate sport-appropriate link based on team ID prefix
		var link string
		switch {
		case len(id) > 5 && id[:5] == "intl_":
			link = fmt.Sprintf("https://www.espncricinfo.com/team/%s", id[5:])
		case len(id) > 4 && id[:4] == "t20_":
			link = fmt.Sprintf("https://www.espncricinfo.com/team/%s", id[4:])
		default:
			link = fmt.Sprintf("https://www.espn.com/mens-college-basketball/team/_/id/%s", id)
		}

		alerts = append(alerts, Alert{
			TeamID:   id,
			Team:     name,
			Event:    liveStatus,
			Link:     link,
			Priority: priority,
		})
	}
	teamCacheMu.RUnlock()

	// State machine: elevate mood to 'alert' if any game is active
	mood := "idle"
	message := fmt.Sprintf("Tracking %d team(s).", len(alerts))
	
	if hasActiveGame {
		mood = "alert"
		message = "Live games in progress!"
	}

	response := StatusResponse{
		Mood:    mood,
		Message: message,
		Alerts:  alerts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// discoveryHandler fetches available sports and teams concurrently from all sources
func discoveryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type ncaaResult struct {
		teams []DiscoveryTeam
		err   error
	}
	type cricketResult struct {
		intl []DiscoveryTeam
		t20  []DiscoveryTeam
		err  error
	}

	ncaaCh := make(chan ncaaResult, 1)
	cricketCh := make(chan cricketResult, 1)

	// Fetch NCAA and Cricket data concurrently
	go func() {
		teams, err := FetchNCAAMBBTeams()
		ncaaCh <- ncaaResult{teams, err}
	}()
	go func() {
		intl, t20, err := FetchCricketTeams()
		cricketCh <- cricketResult{intl, t20, err}
	}()

	ncaa := <-ncaaCh
	cricket := <-cricketCh

	if ncaa.err != nil {
		logger.Error("NCAA discovery error", slog.String("error", ncaa.err.Error()))
	}
	if cricket.err != nil {
		logger.Error("Cricket discovery error", slog.String("error", cricket.err.Error()))
	}

	// If both failed, return an error
	if ncaa.err != nil && cricket.err != nil {
		http.Error(w, "Failed to fetch teams from all sources", http.StatusBadGateway)
		return
	}

	// Build the combined team name cache
	teamCacheMu.Lock()
	teamCache = make(map[string]string)
	if ncaa.err == nil {
		for _, t := range ncaa.teams {
			teamCache[t.ID] = t.Name
		}
	}
	if cricket.err == nil {
		for _, t := range cricket.intl {
			teamCache[t.ID] = t.Name
		}
		for _, t := range cricket.t20 {
			teamCache[t.ID] = t.Name
		}
	}
	teamCacheMu.Unlock()

	// Assemble response — include each sport that succeeded
	var sports []DiscoverySport

	if ncaa.err == nil {
		sports = append(sports, DiscoverySport{
			ID:    "ncaa_mbb",
			Name:  "NCAA Men's Basketball",
			Teams: ncaa.teams,
		})
	}
	if cricket.err == nil {
		sports = append(sports, DiscoverySport{
			ID:    "cricket_intl",
			Name:  "Cricket (International)",
			Teams: cricket.intl,
		})
		sports = append(sports, DiscoverySport{
			ID:    "cricket_t20",
			Name:  "Cricket (T20 Leagues)",
			Teams: cricket.t20,
		})
	}

	response := DiscoveryResponse{Sports: sports}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// configHandler handles GET (read config) and POST (update config)
func configHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		configMu.RLock()
		defer configMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appConfig)

	case http.MethodPost:
		var incoming AppConfig
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		configMu.Lock()
		appConfig.SelectedTeams = incoming.SelectedTeams
		configMu.Unlock()

		saveConfig()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// loadConfig reads the persisted config from disk
func loadConfig() {
	data, err := os.ReadFile(configFileName)
	if err != nil {
		logger.Info("No existing config found, starting fresh", slog.String("error", err.Error()))
		appConfig = AppConfig{SelectedTeams: []string{}}
		return
	}

	if err := json.Unmarshal(data, &appConfig); err != nil {
		logger.Error("Failed to parse config, starting fresh", slog.String("error", err.Error()))
		appConfig = AppConfig{SelectedTeams: []string{}}
		return
	}

	if appConfig.SelectedTeams == nil {
		appConfig.SelectedTeams = []string{}
	}

	logger.Info("Loaded config", slog.Int("selected_teams", len(appConfig.SelectedTeams)))
}

// saveConfig writes the current config to disk
func saveConfig() {
	configMu.RLock()
	data, err := json.MarshalIndent(appConfig, "", "  ")
	configMu.RUnlock()

	if err != nil {
		logger.Error("Failed to marshal config", slog.String("error", err.Error()))
		return
	}

	if err := os.WriteFile(configFileName, data, 0644); err != nil {
		logger.Error("Failed to write config", slog.String("error", err.Error()))
	} else {
		logger.Info("Config saved", slog.Int("selected_teams", len(appConfig.SelectedTeams)))
	}
}

// corsMiddleware handles CORS for local React/Electron apps
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
