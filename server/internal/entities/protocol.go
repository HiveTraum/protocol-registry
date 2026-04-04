package entities

import (
	"time"

	"github.com/google/uuid"
)

type ProtocolType int16

const (
	ProtocolTypeUnspecified ProtocolType = 0
	ProtocolTypeGRPC        ProtocolType = 1
)

func (t ProtocolType) String() string {
	switch t {
	case ProtocolTypeGRPC:
		return "grpc"
	default:
		return "unknown"
	}
}

type Protocol struct {
	ID          uuid.UUID
	ServiceID   uuid.UUID
	Type        ProtocolType
	Version     string
	ContentHash string
	UpdatedAt   time.Time
}
