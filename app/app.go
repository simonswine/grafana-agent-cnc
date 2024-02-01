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

	lck   sync.RWMutex
	Rules []model.Rule

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
