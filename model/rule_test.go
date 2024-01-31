package model_test

import (
	"encoding/json"
	"testing"

	labels "github.com/simonswine/prometheus-labels"
	"github.com/stretchr/testify/require"

	"github.com/simonswine/grafana-agent-cnc/model"
)

func testRule() model.Rule {
	return model.Rule{
		ID:     1234,
		Action: model.ActionDrop,
		Selector: model.Selector{
			labels.MustNewMatcher(labels.MatchEqual, "namespace", "dev"),
		},
	}
}

func TestJSONRoundTrip(t *testing.T) {
	tR := testRule()
	data, err := json.Marshal(&tR)
	require.NoError(t, err)

	require.JSONEq(t, `{
  "action": "drop",
  "id": 1234,
  "selector": [
    [
      "namespace",
      "=",
      "dev"
    ]
  ]
}`, string(data))

	var rule model.Rule
	err = json.Unmarshal(data, &rule)
	require.NoError(t, err)

	tR = testRule()
	require.Equal(t, tR.ID, rule.ID)
	require.Equal(t, tR.Action, rule.Action)
	require.Len(t, rule.Selector, 1)
	require.Equal(t, tR.Selector[0].String(), rule.Selector[0].String())
}
