package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/Kuadrant/mcp-gateway/internal/config"
)

// Broker manages upstream A2A agents and serves a federated Agent Card.
type Broker interface {
	// FederatedCard fetches cards from all registered upstream agents and returns
	// a merged card with prefixed skill IDs, advertising the gateway as the endpoint.
	FederatedCard(ctx context.Context) AgentCard

	// ServeAgentCard handles GET /.well-known/agent.json.
	ServeAgentCard(w http.ResponseWriter, r *http.Request)

	// GetAgentInfo returns the upstream agent config for a given prefixed skill ID.
	GetAgentInfo(skillID string) (*config.A2AAgent, error)

	// SetAgents replaces the registered agent list. Called on config reload.
	SetAgents(agents []*config.A2AAgent)
}

type brokerImpl struct {
	mu         sync.RWMutex
	agents     []*config.A2AAgent
	gatewayURL string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewBroker creates a new Broker.
// gatewayURL is the publicly accessible URL the federated card will advertise.
// An optional *http.Client may be supplied for testing; pass nil to use the default.
func NewBroker(agents []*config.A2AAgent, gatewayURL string, logger *slog.Logger, httpClient *http.Client) Broker {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	cp := make([]*config.A2AAgent, len(agents))
	copy(cp, agents)
	return &brokerImpl{
		agents:     cp,
		gatewayURL: gatewayURL,
		httpClient: httpClient,
		logger:     logger,
	}
}

// FederatedCard fetches Agent Cards from all enabled upstream agents concurrently,
// applies each agent's skill prefix, and merges the results into a single card.
// This mirrors how the MCP broker federates tools from upstream MCP servers.
func (b *brokerImpl) FederatedCard(ctx context.Context) AgentCard {
	b.mu.RLock()
	agents := make([]*config.A2AAgent, len(b.agents))
	copy(agents, b.agents)
	b.mu.RUnlock()

	type result struct {
		card  *AgentCard
		agent *config.A2AAgent
	}
	ch := make(chan result, len(agents))

	for _, agent := range agents {
		if agent == nil || !agent.Enabled {
			ch <- result{agent: agent}
			continue
		}
		go func(a *config.A2AAgent) {
			card, err := b.fetchCard(ctx, a.CardURL)
			if err != nil {
				b.logger.Error("failed to fetch agent card", "agent", a.Name, "url", a.CardURL, "error", err)
				ch <- result{agent: a}
				return
			}
			ch <- result{card: card, agent: a}
		}(agent)
	}

	federated := AgentCard{
		Name:   "MCP Gateway",
		URL:    b.gatewayURL,
		Skills: []Skill{},
	}

	for range agents {
		r := <-ch
		if r.card == nil {
			continue
		}
		for _, skill := range r.card.Skills {
			skill.ID = r.agent.SkillPrefix + skill.ID
			federated.Skills = append(federated.Skills, skill)
		}
	}
	return federated
}

// ServeAgentCard handles GET /.well-known/agent.json.
func (b *brokerImpl) ServeAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	card := b.FederatedCard(r.Context())
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(card); err != nil {
		b.logger.Error("failed to encode agent card", "error", err)
	}
}

// GetAgentInfo returns the upstream agent whose skill prefix is the longest match for skillID.
// Longest-match prevents shorter prefixes from shadowing longer ones.
func (b *brokerImpl) GetAgentInfo(skillID string) (*config.A2AAgent, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var best *config.A2AAgent
	for _, a := range b.agents {
		if a == nil || !a.Enabled || len(a.SkillPrefix) == 0 {
			continue
		}
		if strings.HasPrefix(skillID, a.SkillPrefix) {
			if best == nil || len(a.SkillPrefix) > len(best.SkillPrefix) {
				best = a
			}
		}
	}
	if best != nil {
		return best, nil
	}
	return nil, fmt.Errorf("skill %q does not match any registered agent", skillID)
}

// SetAgents replaces the agent list atomically, copying the slice to avoid
// retaining a reference to caller-owned memory.
func (b *brokerImpl) SetAgents(agents []*config.A2AAgent) {
	cp := make([]*config.A2AAgent, len(agents))
	copy(cp, agents)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.agents = cp
}

func (b *brokerImpl) fetchCard(ctx context.Context, cardURL string) (*AgentCard, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cardURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			b.logger.Error("failed to close response body", "error", cerr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned %d", resp.StatusCode)
	}
	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, err
	}
	return &card, nil
}
