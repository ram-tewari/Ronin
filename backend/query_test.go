package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// setupTestState initializes global state for tests so queryHandler
// can read selectedTeams, teamCache, and liveScoreCache.
func setupTestState() {
	configMu.Lock()
	appConfig = AppConfig{
		SelectedTeams: []string{"2509"},
	}
	configMu.Unlock()

	teamCacheMu.Lock()
	teamCache = map[string]string{
		"2509": "Purdue Boilermakers",
	}
	teamCacheMu.Unlock()

	liveScoreCacheMu.Lock()
	liveScoreCache = map[string]string{
		"2509": "Purdue 72 - Indiana 65 (2nd Half 4:30)",
	}
	liveScoreCacheMu.Unlock()
}

func TestQueryHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		ollamaStatus   int
		ollamaBody     string
		ollamaTimeout  bool
		wantStatus     int
		wantMood       string
		wantMsgContain string
	}{
		{
			name:   "perfect JSON response from Ollama",
			method: http.MethodPost,
			body:   `{"query": "How is Purdue doing?"}`,
			ollamaStatus: http.StatusOK,
			ollamaBody: `{"response": "{\"mood\": \"hyped\", \"message\": \"Purdue leads. Do not lose focus.\", \"link\": \"https://espn.com/game/12345\"}"}`,
			wantStatus:     http.StatusOK,
			wantMood:       "hyped",
			wantMsgContain: "Purdue",
		},
		{
			name:   "malformed JSON from Ollama (hallucination)",
			method: http.MethodPost,
			body:   `{"query": "Tell me about Purdue"}`,
			ollamaStatus: http.StatusOK,
			ollamaBody: `{"response": "Sure! Here is some info about Purdue that is NOT valid JSON at all {{{"}`,
			wantStatus:     http.StatusOK,
			wantMood:       "exhausted",
			wantMsgContain: "incoherent",
		},
		{
			name:          "Ollama timeout (server crash simulation)",
			method:        http.MethodPost,
			body:          `{"query": "What is the score?"}`,
			ollamaTimeout: true,
			wantStatus:    http.StatusOK,
			wantMood:      "exhausted",
			wantMsgContain: "not responding",
		},
		{
			name:         "Ollama returns HTTP 500",
			method:       http.MethodPost,
			body:         `{"query": "Any updates?"}`,
			ollamaStatus: http.StatusInternalServerError,
			ollamaBody:   `{"error": "model not found"}`,
			wantStatus:   http.StatusOK,
			wantMood:     "exhausted",
			wantMsgContain: "error",
		},
		{
			name:       "wrong HTTP method (GET instead of POST)",
			method:     http.MethodGet,
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "empty query body",
			method:     http.MethodPost,
			body:       `{"query": ""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON body",
			method:     http.MethodPost,
			body:       `not json at all`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "Ollama returns invalid mood, defaults to idle",
			method: http.MethodPost,
			body:   `{"query": "test"}`,
			ollamaStatus: http.StatusOK,
			ollamaBody: `{"response": "{\"mood\": \"angry\", \"message\": \"Sand coffin.\", \"link\": \"\"}"}`,
			wantStatus:     http.StatusOK,
			wantMood:       "idle",
			wantMsgContain: "Sand coffin",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupTestState()

			// Set up a fake Ollama server
			if tc.ollamaTimeout {
				// Create a server that hangs until the context is cancelled
				fakeOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Simulate a crash by sleeping longer than the timeout
					time.Sleep(5 * time.Second)
				}))
				defer fakeOllama.Close()
				OllamaEndpoint = fakeOllama.URL
				OllamaTimeout = 100 * time.Millisecond // very short timeout for test
			} else if tc.ollamaStatus != 0 {
				fakeOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify the request is well-formed
					if r.Method != http.MethodPost {
						t.Errorf("expected POST to Ollama, got %s", r.Method)
					}
					if ct := r.Header.Get("Content-Type"); ct != "application/json" {
						t.Errorf("expected Content-Type application/json, got %s", ct)
					}

					// Verify the request body contains our model and prompt
					var ollamaReq OllamaRequest
					if err := json.NewDecoder(r.Body).Decode(&ollamaReq); err != nil {
						t.Errorf("failed to decode Ollama request: %v", err)
					}
					if ollamaReq.Model != "llama3.1:8b" {
						t.Errorf("expected model llama3.1:8b, got %s", ollamaReq.Model)
					}
					if ollamaReq.Stream != false {
						t.Errorf("expected stream=false")
					}
					if ollamaReq.Format != "json" {
						t.Errorf("expected format=json, got %s", ollamaReq.Format)
					}
					// The prompt should contain the injected context
					if !strings.Contains(ollamaReq.Prompt, "Purdue Boilermakers") {
						t.Errorf("prompt should contain team name, got: %s", ollamaReq.Prompt)
					}
					if !strings.Contains(ollamaReq.Prompt, "72") {
						t.Errorf("prompt should contain live score data")
					}

					w.WriteHeader(tc.ollamaStatus)
					w.Write([]byte(tc.ollamaBody))
				}))
				defer fakeOllama.Close()
				OllamaEndpoint = fakeOllama.URL
				OllamaTimeout = 5 * time.Second
			}

			// Build the request to our queryHandler
			req := httptest.NewRequest(tc.method, "/query", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			queryHandler(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d. body: %s", rr.Code, tc.wantStatus, rr.Body.String())
			}

			// For non-200 responses, we only check the status code
			if tc.wantStatus != http.StatusOK {
				return
			}

			var resp QueryResponse
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Mood != tc.wantMood {
				t.Errorf("mood = %q, want %q", resp.Mood, tc.wantMood)
			}

			if tc.wantMsgContain != "" && !strings.Contains(resp.Message, tc.wantMsgContain) {
				t.Errorf("message = %q, want it to contain %q", resp.Message, tc.wantMsgContain)
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	selectedTeams := []string{"2509"}
	teamNames := map[string]string{"2509": "Purdue Boilermakers"}
	liveScores := map[string]string{"2509": "Purdue 72 - Indiana 65"}

	prompt := buildPrompt("How is Purdue doing?", selectedTeams, liveScores, teamNames)

	checks := []string{
		"Ronin",
		"Gaara",
		"ECE 270",
		"Pharos",
		"Purdue Boilermakers",
		"Purdue 72 - Indiana 65",
		"How is Purdue doing?",
		`"mood"`,
		`"message"`,
		`"link"`,
	}

	for _, want := range checks {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPromptNoTeams(t *testing.T) {
	prompt := buildPrompt("hello", nil, nil, nil)

	if !strings.Contains(prompt, "not tracking any teams") {
		t.Errorf("prompt should mention no tracked teams when selectedTeams is empty")
	}
}
