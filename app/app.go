package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/simonswine/grafana-agent-cnc/model"
	labels "github.com/simonswine/prometheus-labels"
)

type App struct {
	wg sync.WaitGroup

	lck        sync.RWMutex
	Rules      []model.Rule
	nextRuleID int64

	upgrader websocket.Upgrader
	hub      *Hub
}

func (a *App) internalError(ws *websocket.Conn, msg string, err error) {
	slog.Error(msg, "err", err)
	ws.WriteMessage(websocket.TextMessage, []byte("Internal server error."))
}

func (a *App) handleRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(a.Rules); err != nil {
		a.internalError(nil, "", err)
		return
	}
}

func (a *App) insertRule(data *model.PayloadRuleInsert) {
	a.lck.Lock()
	defer a.lck.Unlock()

	if a.nextRuleID == 0 {
		for _, r := range a.Rules {
			if r.ID > a.nextRuleID {
				a.nextRuleID = r.ID
			}
		}
		a.nextRuleID++
	}

	pos := 0
	// find position to insert
	if data.After != nil {
		after := *data.After
		for i, r := range a.Rules {
			if r.ID == after {
				pos = i + 1
				break
			}
		}
	}

	// overwrite id
	rule := data.Rule
	rule.ID = a.nextRuleID
	a.nextRuleID++

	// insert rule to correct position
	a.Rules = append(a.Rules, model.Rule{})
	copy(a.Rules[pos+1:], a.Rules[pos:])
	a.Rules[pos] = rule

}

func (a *App) deleteRule(id int64) {
	a.lck.Lock()
	defer a.lck.Unlock()

	for i, r := range a.Rules {
		if r.ID == id {
			a.Rules = append(a.Rules[:i], a.Rules[i+1:]...)
			break
		}
	}
}

func (a *App) getRules() ([]byte, error) {
	a.lck.RLock()
	defer a.lck.RUnlock()
	b, err := json.Marshal(a.Rules)
	slog.Debug("getRules", "rules", string(b))
	return b, err
}

func (a *App) Run(ctx context.Context, args ...string) error {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.hub.run()
	}()
	port := ":8333"
	http.HandleFunc("/", a.handleRules)
	http.HandleFunc("/ws", a.handleWS)
	http.HandleFunc("/ws/ui", a.handleWS)
	http.HandleFunc("/ws/grafana-agent", a.handleWS)
	return http.ListenAndServe(port, nil)
}

func New() *App {
	a := &App{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true }, // TODO: check origin
		},
		Rules: []model.Rule{
			{
				ID: 1234,
				Selector: model.Selector{
					labels.MustNewMatcher(labels.MatchEqual, "namespace", "dev"),
					labels.MustNewMatcher(labels.MatchEqual, "name", "test"),
				},
				Action: model.ActionDrop,
			},
			{
				ID:     5678,
				Action: model.ActionKeep,
			},
		},
	}

	a.hub = newHub(
		a,
		newTopic("rules", a.getRules),
		newTopic("agents", func() ([]byte, error) {
			if a.hub == nil {
				return []byte{}, nil
			}
			return a.hub.getAgents()
		}),
	)
	return a
}
