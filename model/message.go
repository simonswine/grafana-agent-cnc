package model

type MessageType string

const (
	MessageTypeUnknown   MessageType = "unknown"
	MessageTypeSubscribe MessageType = "subscribe"
	MessageTypeData      MessageType = "data"
)

type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

type PayloadData struct {
	Rules  []Rule  `json:"rules"`
	Agents []Agent `json:"agents"`
}

type PayloadSubscribe struct {
	Topics []string `json:"topics"`
}
