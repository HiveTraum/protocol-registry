package publish_protocol

import "github.com/user/protocol_registry/internal/entities"

type Input struct {
	ServiceName  string
	ProtocolType entities.ProtocolType
	FileSet      entities.ProtoFileSet
	Versions     []string
}

type Output struct {
	ServiceName string
	IsNew       bool
}
