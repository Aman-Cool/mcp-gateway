package a2a

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Kuadrant/mcp-gateway/internal/config"
	"github.com/stretchr/testify/require"
)

func TestFederatedCard_mergesSkillsWithPrefix(t *testing.T) {
	weather := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(AgentCard{ //nolint:errcheck
			Name: "Weather Agent",
			URL:  "http://weather.internal",
			Skills: []Skill{
				{ID: "forecast", Name: "Get Forecast", Description: "Returns a weather forecast"},
				{ID: "alerts", Name: "Get Alerts"},
			},
		})
	}))
	defer weather.Close()

	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(AgentCard{ //nolint:errcheck
			Name: "Search Agent",
			URL:  "http://search.internal",
			Skills: []Skill{
				{ID: "web_search", Name: "Web Search"},
			},
		})
	}))
	defer search.Close()

	agents := []*config.A2AAgent{
		{Name: "weather", CardURL: weather.URL, SkillPrefix: "weather_", Enabled: true},
		{Name: "search", CardURL: search.URL, SkillPrefix: "search_", Enabled: true},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	broker := NewA2ABroker(agents, "http://gateway.example.com", logger, nil)

	card := broker.FederatedCard(context.Background())

	require.Equal(t, "MCP Gateway", card.Name)
	require.Equal(t, "http://gateway.example.com", card.URL)
	require.Len(t, card.Skills, 3)

	skillIDs := make(map[string]string, len(card.Skills))
	for _, s := range card.Skills {
		skillIDs[s.ID] = s.Name
	}
	require.Equal(t, "Get Forecast", skillIDs["weather_forecast"])
	require.Equal(t, "Get Alerts", skillIDs["weather_alerts"])
	require.Equal(t, "Web Search", skillIDs["search_web_search"])
}

func TestFederatedCard_skipsDisabledAgents(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		json.NewEncoder(w).Encode(AgentCard{Skills: []Skill{{ID: "x"}}}) //nolint:errcheck
	}))
	defer srv.Close()

	agents := []*config.A2AAgent{
		{Name: "disabled", CardURL: srv.URL, SkillPrefix: "d_", Enabled: false},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	broker := NewA2ABroker(agents, "http://gateway.example.com", logger, nil)
	card := broker.FederatedCard(context.Background())

	require.False(t, called, "disabled agent should not be fetched")
	require.Empty(t, card.Skills)
}

func TestFederatedCard_toleratesUpstreamFailure(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(AgentCard{Skills: []Skill{{ID: "ok", Name: "OK"}}}) //nolint:errcheck
	}))
	defer good.Close()

	agents := []*config.A2AAgent{
		{Name: "good", CardURL: good.URL, SkillPrefix: "g_", Enabled: true},
		{Name: "bad", CardURL: "http://127.0.0.1:1", SkillPrefix: "b_", Enabled: true},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	broker := NewA2ABroker(agents, "http://gateway.example.com", logger, nil)
	card := broker.FederatedCard(context.Background())

	require.Len(t, card.Skills, 1)
	require.Equal(t, "g_ok", card.Skills[0].ID)
}

func TestGetAgentInfo_resolvesByPrefix(t *testing.T) {
	agents := []*config.A2AAgent{
		{Name: "weather", CardURL: "http://weather.internal", SkillPrefix: "weather_", Enabled: true},
		{Name: "search", CardURL: "http://search.internal", SkillPrefix: "search_", Enabled: true},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	broker := NewA2ABroker(agents, "http://gateway.example.com", logger, nil)

	agent, err := broker.GetAgentInfo("weather_forecast")
	require.NoError(t, err)
	require.Equal(t, "weather", agent.Name)

	agent, err = broker.GetAgentInfo("search_web_search")
	require.NoError(t, err)
	require.Equal(t, "search", agent.Name)

	_, err = broker.GetAgentInfo("unknown_skill")
	require.Error(t, err)
}

func TestServeAgentCard_methodNotAllowed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	broker := NewA2ABroker(nil, "http://gateway.example.com", logger, nil)

	w := httptest.NewRecorder()
	broker.ServeAgentCard(w, httptest.NewRequest(http.MethodPost, "/.well-known/agent.json", nil))
	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
