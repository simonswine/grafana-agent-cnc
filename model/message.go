package model

type MessageType string

const (
	MessageTypeUnknown    MessageType = "unknown"
	MessageTypeSubscribe  MessageType = "subscribe"
	MessageTypeData       MessageType = "data"
	MessageTypeRuleInsert MessageType = "rule.insert"
	MessageTypeRuleDelete MessageType = "rule.delete"
)

type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

type PayloadSubscribe struct {
	Topics []string `json:"topics"`
}

type PayloadData struct {
	Rules  []Rule  `json:"rules,omitempty"`
	Agents []Agent `json:"agents,omitempty"`
}

type PayloadRuleInsert struct {
	Rule  Rule   `json:"rule,omitempty"`
	After *int64 `json:"agents,omitempty"`
}

type PayloadRuleDelete struct {
	ID *int64 `json:"id,omitempty"`
}
