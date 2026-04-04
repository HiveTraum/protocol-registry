package validate_protocol

import "github.com/user/protocol_registry/internal/entities"

type Input struct {
	ServiceName     string
	ProtocolType    entities.ProtocolType
	FileSet         entities.ProtoFileSet
	AgainstVersions []string
}

type VersionViolation struct {
	Version   string
	Consumers []entities.ConsumerBreakingChange
}

type Output struct {
	Valid      bool
	Violations []VersionViolation
}
