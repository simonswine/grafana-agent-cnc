package model

import "time"

type Agent struct {
	Name        string              `json:"name,omitempty"`
	Targets     []map[string]string `json:"targets,omitempty"`
	LastUpdated time.Time           `json:"last_updated,omitempty"`
}
