package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// QueryRequest is the incoming user query
type QueryRequest struct {
	Query string `json:"query"`
}

// QueryResponse is the structured LLM output
type QueryResponse struct {
	Mood    string `json:"mood"`
	Message string `json:"message"`
	Link    string `json:"link"`
}

// OllamaRequest is the payload sent to the Ollama /api/generate endpoint
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

// OllamaResponse is the response from Ollama /api/generate
type OllamaResponse struct {
	Response string `json:"response"`
}

// OllamaEndpoint is the URL for the local Ollama server. Exported for testing.
var OllamaEndpoint = "http://localhost:11434/api/generate"

// OllamaTimeout controls how long we wait for Ollama to respond.
var OllamaTimeout = 90 * time.Second

// buildPrompt constructs the system + context prompt for the Ronin persona.
func buildPrompt(userQuery string, selectedTeams []string, liveScores map[string]string, teamNames map[string]string) string {
	var sb strings.Builder

	sb.WriteString(`You are Ronin, a desktop assistant with the personality of Gaara from Naruto. `)
	sb.WriteString(`You are stern, highly focused on the user's productivity, and speak with intense, calm brevity. No emojis. Ever. `)
	sb.WriteString(`You are fiercely protective of the user's focus. `)
	sb.WriteString(`If the user asks a stupid question, tries to distract themselves, or there are no active games happening, `)
	sb.WriteString(`you must brutally remind them to get back to studying for ECE 270 and building the Pharos architecture. `)
	sb.WriteString(`Do not be friendly. Do not be encouraging. Be direct and ruthless like Gaara in the Chunin Exams.`)
	sb.WriteString("\n\n")

	// Inject live context
	sb.WriteString("=== LIVE SPORTS CONTEXT ===\n")
	if len(selectedTeams) == 0 {
		sb.WriteString("The user is not tracking any teams.\n")
	} else {
		for _, id := range selectedTeams {
			name := teamNames[id]
			if name == "" {
				name = "Team " + id
			}
			score := liveScores[id]
			if score == "" {
				score = "No data"
			}
			sb.WriteString(fmt.Sprintf("- %s [%s]: %s\n", name, id, score))
		}
	}
	sb.WriteString("=== END CONTEXT ===\n\n")

	sb.WriteString("User's question: ")
	sb.WriteString(userQuery)
	sb.WriteString("\n\n")

	sb.WriteString(`You MUST respond with a single JSON object. No extra text, no markdown, no explanation. `)
	sb.WriteString(`The JSON must have exactly these fields:
{
  "mood": one of "idle", "alert", "hyped", or "exhausted",
  "message": your response as Ronin (1-2 sentences max),
  "link": a valid URL to a relevant match thread from the live context above, or an empty string if none applies
}`)

	return sb.String()
}

// gatherContext reads the current selected teams, team names, and live scores.
func gatherContext() (selectedTeams []string, teamNames map[string]string, liveScores map[string]string) {
	configMu.RLock()
	selectedTeams = make([]string, len(appConfig.SelectedTeams))
	copy(selectedTeams, appConfig.SelectedTeams)
	configMu.RUnlock()

	teamCacheMu.RLock()
	teamNames = make(map[string]string, len(teamCache))
	for k, v := range teamCache {
		teamNames[k] = v
	}
	teamCacheMu.RUnlock()

	liveScoreCacheMu.RLock()
	liveScores = make(map[string]string, len(liveScoreCache))
	for k, v := range liveScoreCache {
		liveScores[k] = v
	}
	liveScoreCacheMu.RUnlock()

	return
}

// queryHandler handles POST /query — sends the user's question to the local Ollama LLM
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

	query := strings.TrimSpace(req.Query)

	// Gather live context
	selectedTeams, teamNames, liveScores := gatherContext()

	// Build the prompt with injected context
	prompt := buildPrompt(query, selectedTeams, liveScores, teamNames)

	// Prepare Ollama request
	ollamaReq := OllamaRequest{
		Model:  "llama3.1:8b",
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		http.Error(w, "Failed to build LLM request", http.StatusInternalServerError)
		return
	}

	// Call the local Ollama server with a timeout
	ctx, cancel := context.WithTimeout(r.Context(), OllamaTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, OllamaEndpoint, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "Failed to create LLM request", http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		// Timeout or connection refused — Ollama is down
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(QueryResponse{
			Mood:    "exhausted",
			Message: "The sand is silent. Ollama is not responding.",
			Link:    "",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(QueryResponse{
			Mood:    "exhausted",
			Message: "The sand stirs but cannot form. Ollama returned an error.",
			Link:    "",
		})
		return
	}

	// Parse Ollama's response
	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(QueryResponse{
			Mood:    "exhausted",
			Message: "Ollama spoke, but the sand could not interpret it.",
			Link:    "",
		})
		return
	}

	// Parse the LLM's JSON output into our QueryResponse
	var queryResp QueryResponse
	if err := json.Unmarshal([]byte(ollamaResp.Response), &queryResp); err != nil {
		// LLM hallucinated malformed JSON — return a graceful fallback
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(QueryResponse{
			Mood:    "exhausted",
			Message: "The sand crumbles. The response was... incoherent. Focus on your work.",
			Link:    "",
		})
		return
	}

	// Validate mood field
	switch queryResp.Mood {
	case "idle", "alert", "hyped", "exhausted":
		// valid
	default:
		queryResp.Mood = "idle"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queryResp)
}
