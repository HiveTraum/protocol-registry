package list_protocol_versions

import (
	"time"

	"github.com/user/protocol_registry/internal/entities"
)

type Input struct {
	ServiceName  string
	ProtocolType entities.ProtocolType
	Offset       int
	Limit        int
}

type Output struct {
	ServiceName  string
	ProtocolType entities.ProtocolType
	Versions     []VersionInfo
	Total        int
}

type VersionInfo struct {
	VersionNumber int
	ContentHash   string
	FileCount     int
	PublishedAt   time.Time
}
