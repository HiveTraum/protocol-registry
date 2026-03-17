package register_consumer

import "github.com/user/protocol_registry/internal/entities"

type Input struct {
	ConsumerName string
	ServerName   string
	ProtocolType entities.ProtocolType
	FileSet      entities.ProtoFileSet
}

type Output struct {
	ConsumerName string
	ServerName   string
	IsNew        bool
}
