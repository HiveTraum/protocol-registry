package get_protocol

import "github.com/user/protocol_registry/internal/entities"

type Input struct {
	ServiceName  string
	ProtocolType entities.ProtocolType
	Version      string
}

type Output struct {
	ServiceName  string
	ProtocolType entities.ProtocolType
	Version      string
	FileSet      entities.ProtoFileSet
}
