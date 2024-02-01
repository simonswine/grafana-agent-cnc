package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/cespare/xxhash/v2"

	"github.com/simonswine/grafana-agent-cnc/model"
)

type topic struct {
	name    string
	ch      chan struct{}
	update  func() ([]byte, error)
	err     error
	content []byte
	hasher  xxhash.Digest
	hash    uint64
	clients map[*Client]uint64
}

func newTopic(name string, f func() ([]byte, error)) *topic {
	return &topic{
		name:    name,
		ch:      make(chan struct{}),
		update:  f,
		hash:    0,
		clients: make(map[*Client]uint64),
	}
}

func (t *topic) get() {
	t.content, t.err = t.update()
	if t.err != nil {
		return
	}

	t.hasher.Reset()
	_, t.err = t.hasher.Write(t.content)
	if t.err != nil {
		return
	}
	t.hash = t.hasher.Sum64()
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	topics map[string]*topic

	agentsPublishing       map[*Client]bool // are particular grafana agents publishing their targets
	agentsPublishingActive bool             // is agent publishing requested
	agents                 map[string]*model.Agent

	// Register requests from the clients.
	registerCh chan *Client

	// Unregister requests from clients.
	unregisterCh chan *Client

	// Update agent targets
	agentCh chan *model.Agent
}

func newHub(topics ...*topic) *Hub {
	t := make(map[string]*topic)
	for _, topic := range topics {
		t[topic.name] = topic
	}
	return &Hub{
		topics:           t,
		agentsPublishing: make(map[*Client]bool),
		registerCh:       make(chan *Client),
		unregisterCh:     make(chan *Client),
	}
}

// check data for topics to sent
func (h *Hub) updateClientTopicsToPublish(client *Client) error {
	var payload = make(map[string]json.RawMessage)
	for tname, t := range h.topics {
		h, ok := t.clients[client]
		if !ok {
			continue
		}

		if t.hash == 0 {
			t.get()
		}

		if h == t.hash {
			continue
		}

		// need update
		payload[tname] = t.content
		t.clients[client] = t.hash
	}

	if len(payload) > 0 {
		data, err := json.Marshal(&model.Message{
			Type:    model.MessageTypeData,
			Payload: payload,
		})
		if err != nil {
			return fmt.Errorf("error generation JSON: %w", err)
		}
		client.send <- data
	}
	return nil
}

// check if agent needs toggle publishing
func (h *Hub) updateClientToggleAgentSubscription(client *Client) error {
	active, ok := h.agentsPublishing[client]
	if !ok {
		return nil
	}
	if active != h.agentsPublishingActive {
		msg := model.PayloadSubscribe{}
		if !active {
			msg.Topics = []string{"agents"}
		}
		data, err := json.Marshal(&model.Message{
			Type:    model.MessageTypeSubscribe,
			Payload: msg,
		})
		if err != nil {
			return err
		}

		slog.Debug("request agent publishing", "data", string(data))
		client.send <- data

		h.agentsPublishing[client] = h.agentsPublishingActive
	}
	return nil
}

func (h *Hub) updateClient(client *Client) {
	if err := h.updateClientTopicsToPublish(client); err != nil {
		slog.Error("error updating client topics to publish", "err", err)
	}
	if err := h.updateClientToggleAgentSubscription(client); err != nil {
		slog.Error("error updating client to subscribe to", "err", err)
	}
}

func (h *Hub) updateAgents() {
	t, ok := h.topics["agents"]
	if !ok {
		return
	}
	t.get()

	// update frontend with new agents data
	for client := range t.clients {
		h.updateClient(client)
	}
}

func (h *Hub) getAgents() ([]byte, error) {
	var (
		agentNames = make([]string, len(h.agents))
		agents     = make([]*model.Agent, len(h.agents))
	)

	for name := range h.agents {
		agentNames = append(agentNames, name)
	}
	sort.Strings(agentNames)

	for i, name := range agentNames {
		agents[i] = h.agents[name]
	}

	return json.Marshal(agents)
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.registerCh:
			for _, topic := range client.subscribedTopics {
				t, ok := h.topics[topic]
				if !ok {
					continue
				}
				_, ok = t.clients[client]
				if ok {
					continue
				}
				t.clients[client] = 0

			}
			if client.isGrafanaAgent {
				h.agentsPublishing[client] = false
			}

			// check if agents need toggle publishing
			agentsTopic, ok := h.topics["agents"]
			agentsPublishingRequested := false
			if ok {
				agentsPublishingRequested = len(agentsTopic.clients) != 0
			}
			if agentsPublishingRequested != h.agentsPublishingActive {
				if agentsPublishingRequested {
					slog.Debug("agents publishing has been enabled")
				} else {
					slog.Debug("agents publishing has been disabled")
				}
				h.agentsPublishingActive = agentsPublishingRequested
				// send message to all agents
				for agent := range h.agentsPublishing {
					h.updateClient(agent)
				}
			}
			h.updateClient(client)
		case agent := <-h.agentCh:
			agent.LastUpdated = time.Now()
			h.agents[agent.Name] = agent
			h.updateAgents()
		case client := <-h.unregisterCh:
			for _, topic := range h.topics {
				delete(topic.clients, client)
			}
		}
	}
}
