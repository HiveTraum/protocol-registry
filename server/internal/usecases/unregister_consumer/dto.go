package unregister_consumer

import "github.com/user/protocol_registry/internal/entities"

type Input struct {
	ConsumerName string
	ServerName   string
	ProtocolType entities.ProtocolType
}
