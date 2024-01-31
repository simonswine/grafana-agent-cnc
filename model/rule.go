package model

import (
	"encoding/json"

	labels "github.com/simonswine/prometheus-labels"
)

type Action uint8

const (
	ActionUndefined Action = iota
	ActionKeep
	ActionDrop
	ActionContinue
)

func (a Action) String() string {
	switch a {
	case ActionKeep:
		return "keep"
	case ActionDrop:
		return "drop"
	case ActionContinue:
		return "continue"
	default:
		return "undefined"
	}
}

func (a Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

type Selector labels.Selector

func (s Selector) MarshalJSON() ([]byte, error) {
	res := make([]string, len(s))
	for idx := range s {
		res[idx] = s[idx].String()
	}
	return json.Marshal(res)
}

type Rule struct {
	ID       int64
	Selector Selector
	Action   Action
}
