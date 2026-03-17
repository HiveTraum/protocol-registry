package get_protocol

import "github.com/user/protocol_registry/internal/entities"

type Input struct {
	ServiceName  string
	ProtocolType entities.ProtocolType
}

type Output struct {
	ServiceName  string
	ProtocolType entities.ProtocolType
	FileSet      entities.ProtoFileSet
}
