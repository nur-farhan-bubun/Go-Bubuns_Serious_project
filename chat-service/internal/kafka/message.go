package kafka

import (
	"encoding/json"

	"github.com/ride-sharing/chat-service/internal/domain"
)

// Message is the structure stored in Kafka. It carries enough context for
// any service instance to reconstruct the WSEnvelope and fan it out to its
// locally-connected WebSocket clients.
type Message struct {
	RoomID string              `json:"room_id"`
	Type   domain.WSMessageType `json:"type"`
	Data   json.RawMessage     `json:"data"`
}
