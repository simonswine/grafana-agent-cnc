package app

import (
	"encoding/json"
	"log/slog"

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

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub(topics ...*topic) *Hub {
	t := make(map[string]*topic)
	for _, topic := range topics {
		t[topic.name] = topic
	}
	return &Hub{
		topics:     t,
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}
func (h *Hub) updateClient(client *Client) {
	var payload = make(map[string]json.RawMessage)
	for tname, t := range h.topics {
		h, ok := t.clients[client]
		if !ok {
			continue
		}

		if t.hash == 0 {
			slog.Debug("topic %s not initialized", "topic", tname)
			t.get()
		}

		if h == t.hash {
			continue
		}

		// need update
		payload[tname] = t.content
		t.clients[client] = t.hash
	}

	data, err := json.Marshal(&model.Message{
		Type:    model.MessageTypeData,
		Payload: payload,
	})
	if err != nil {
		panic(err)
	}
	client.send <- data
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
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
			h.updateClient(client)
		case client := <-h.unregister:
			for _, topic := range h.topics {
				delete(topic.clients, client)
			}
		}
	}
}
