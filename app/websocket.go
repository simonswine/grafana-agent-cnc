package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/gorilla/websocket"
	"github.com/simonswine/grafana-agent-cnc/model"
)

var (
	newline = []byte{'\n'}
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub    *Hub
	logger *slog.Logger

	// The websocket connection.
	conn *websocket.Conn

	isGrafanaAgent   bool
	subscribedTopics []string

	// Buffered channel of outbound messages.
	send chan []byte
}

type recvMsg struct {
	Type       model.MessageType `json:"type"`
	Data       *model.PayloadData
	Subscribe  *model.PayloadSubscribe
	RuleDelete *model.PayloadRuleDelete
	RuleInsert *model.PayloadRuleInsert
}

func (m *recvMsg) UnmarshalJSON(b []byte) error {
	var header struct {
		Type    model.MessageType `json:"type"`
		Payload json.RawMessage   `json:"payload"`
	}
	json.Unmarshal(b, &header)
	m.Type = header.Type
	m.Data = nil
	m.Subscribe = nil

	switch m.Type {
	case model.MessageTypeData:
		var data model.PayloadData
		if err := json.Unmarshal(header.Payload, &data); err != nil {
			return err
		}
		m.Data = &data
	case model.MessageTypeSubscribe:
		var subscribe model.PayloadSubscribe
		if err := json.Unmarshal(header.Payload, &subscribe); err != nil {
			return err
		}
		m.Subscribe = &subscribe
	case model.MessageTypeRuleInsert:
		var e model.PayloadRuleInsert
		if err := json.Unmarshal(header.Payload, &e); err != nil {
			return err
		}
		m.RuleInsert = &e
	case model.MessageTypeRuleDelete:
		var e model.PayloadRuleDelete
		if err := json.Unmarshal(header.Payload, &e); err != nil {
			return err
		}
		m.RuleDelete = &e
	default:
		return fmt.Errorf("unknown message type %s", m.Type)
	}
	return nil

}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregisterCh <- c
		c.conn.Close()
	}()
	var (
		msg recvMsg
	)

	for {
		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("error reading json", "error", err)
			}
			break
		}

		switch msg.Type {
		case model.MessageTypeSubscribe:
			c.subscribedTopics = msg.Subscribe.Topics
			slog.Debug("client subscribing", "topics", msg.Subscribe.Topics)
			c.hub.registerCh <- c
		case model.MessageTypeData:
			for idx := range msg.Data.Agents {
				a := msg.Data.Agents[idx]
				slog.Debug("received agent targets", "agent", a.Name, "targets", len(a.Targets))
				c.hub.agentCh <- &a
			}
		case model.MessageTypeRuleDelete:
			if msg.RuleDelete.ID == nil {
				slog.Warn("received rule delete without id")
				continue
			}
			if c.isGrafanaAgent {
				slog.Warn("grafana-agent is not allowed to delete rules")
				continue
			}
			slog.Debug("received rule delete", "id", msg.RuleDelete.ID)
			id := *msg.RuleDelete.ID
			c.hub.rulesCh <- func(a *App) {
				a.deleteRule(id)
			}
		case model.MessageTypeRuleInsert:
			if c.isGrafanaAgent {
				slog.Warn("grafana-agent is not allowed to insert rules")
				continue
			}
			slog.Debug("received rule insert", "rule", msg.RuleInsert.Rule)
			c.hub.rulesCh <- func(a *App) {
				a.insertRule(msg.RuleInsert)
			}

		default:
			slog.Warn("unknown message type", "type", msg.Type)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			messageLen, _ := w.Write(message)
			if len(message) > 64 {
				message = append(message[:64], []byte("...")...)
			}
			c.logger.Debug("sent message to client", "size", messageLen, "message", message)

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func (a *App) handleWS(w http.ResponseWriter, r *http.Request) {

	conn, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("error upgrading websocket", "err", err)
		return
	}
	client := &Client{
		hub:  a.hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
	client.logger = slog.With("remote", r.RemoteAddr, "user-agent", r.Header.Get("user-agent"), "client", fmt.Sprintf("%p", client))
	client.logger.Debug("new websocket client")

	if filepath.Base(r.URL.Path) == "grafana-agent" {
		client.isGrafanaAgent = true
	}

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
