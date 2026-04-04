package entities

import (
	"time"

	"github.com/google/uuid"
)

type Consumer struct {
	ID                uuid.UUID
	ConsumerServiceID uuid.UUID
	ServerServiceID   uuid.UUID
	ProtocolType      ProtocolType
	Version           string
	ContentHash       string
	UpdatedAt         time.Time
}
