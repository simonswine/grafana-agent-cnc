package model

import (
	"encoding/json"
	"fmt"

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

func (a *Action) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	switch s {
	case "keep":
		*a = ActionKeep
	case "drop":
		*a = ActionDrop
	case "continue":
		*a = ActionContinue
	default:
		*a = ActionUndefined
	}

	return nil
}

type Selector labels.Selector

type matcher [3]string

func (m matcher) Name() string {
	return m[0]
}

func (m matcher) Value() string {
	return m[2]
}

func (m matcher) Type() (labels.MatchType, error) {
	return matchTypeFromString(m[1])
}

func (m matcher) Matcher() (*labels.Matcher, error) {
	t, err := matchTypeFromString(m[1])
	if err != nil {
		return nil, err
	}
	return labels.NewMatcher(t, m[0], m[2])
}

func (s Selector) MarshalJSON() ([]byte, error) {
	res := make([]matcher, len(s))
	for idx := range s {
		res[idx][0] = s[idx].Name
		res[idx][1] = s[idx].Type.String()
		res[idx][2] = s[idx].Value
	}
	return json.Marshal(res)
}

func matchTypeFromString(s string) (labels.MatchType, error) {
	if s == "=" {
		return labels.MatchEqual, nil
	}
	if s == "!=" {
		return labels.MatchNotEqual, nil
	}
	if s == "=~" {
		return labels.MatchRegexp, nil
	}
	if s == "!~" {
		return labels.MatchNotRegexp, nil
	}
	return labels.MatchType(0), fmt.Errorf("unknown match type")
}

func (s *Selector) UnmarshalJSON(b []byte) error {
	matchers := make([][3]string, 0)
	err := json.Unmarshal(b, &matchers)
	if err != nil {
		return err
	}

	selector := make(Selector, 0, len(matchers))
	for _, matcher := range matchers {
		t, err := matchTypeFromString(matcher[1])
		if err != nil {
			return err
		}

		m, err := labels.NewMatcher(t, matcher[0], matcher[2])
		if err != nil {
			return err
		}
		selector = append(selector, m)
	}

	*s = selector

	return nil
}

type Rule struct {
	ID       int64    `json:"id" river:"id,attr"`
	Selector Selector `json:"selector" river:"selector,attr"`
	Action   Action   `json:"action" river:"action,attr"`
}
