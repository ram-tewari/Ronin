package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// QueryRequest is the incoming user query
type QueryRequest struct {
	Query string `json:"query"`
}

// QueryResponse is the mood + message reply
type QueryResponse struct {
	Mood    string `json:"mood"`
	Message string `json:"message"`
}

// queryHandler handles POST /query — matches user text against tracked teams
func queryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Query) == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	queryLower := strings.ToLower(strings.TrimSpace(req.Query))

	// Get current tracked teams
	configMu.RLock()
	selected := make([]string, len(appConfig.SelectedTeams))
	copy(selected, appConfig.SelectedTeams)
	configMu.RUnlock()

	// Build a set of tracked team names (lowercased) from the cache
	teamCacheMu.RLock()
	trackedNames := make(map[string]string) // lowercased name -> original name
	for _, id := range selected {
		if name, ok := teamCache[id]; ok {
			trackedNames[strings.ToLower(name)] = name
		}
	}
	teamCacheMu.RUnlock()

	// Try to find a tracked team mentioned in the query
	var matchedName string
	for nameLower, nameOriginal := range trackedNames {
		if strings.Contains(queryLower, nameLower) {
			matchedName = nameOriginal
			break
		}
	}

	var resp QueryResponse

	if matchedName == "" {
		// No tracked team found in query
		resp = QueryResponse{
			Mood:    "exhausted",
			Message: "You aren't even tracking them. Focus on your code.",
		}
	} else {
		// Mock state-machine logic based on simple keyword heuristics
		if containsAny(queryLower, []string{"win", "won", "score", "result", "update", "how did"}) {
			resp = QueryResponse{
				Mood:    "hyped",
				Message: matchedName + " secured the W. Absolute dominance.",
			}
		} else if containsAny(queryLower, []string{"next", "when", "upcoming", "schedule", "match", "game", "playing"}) {
			resp = QueryResponse{
				Mood:    "alert",
				Message: matchedName + " match incoming in 2 hours. Be ready.",
			}
		} else {
			// Generic query about a tracked team — default to hyped
			resp = QueryResponse{
				Mood:    "hyped",
				Message: matchedName + " secured the W. Absolute dominance.",
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// containsAny returns true if s contains any of the substrings
func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
