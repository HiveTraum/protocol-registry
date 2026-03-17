package entities

import "strings"

type ProtocolType int

const (
	ProtocolTypeUnspecified ProtocolType = 0
	ProtocolTypeGRPC       ProtocolType = 1
)

func ParseProtocolType(s string) ProtocolType {
	switch strings.ToLower(s) {
	case "grpc":
		return ProtocolTypeGRPC
	default:
		return ProtocolTypeUnspecified
	}
}

func (t ProtocolType) String() string {
	switch t {
	case ProtocolTypeGRPC:
		return "grpc"
	default:
		return "unknown"
	}
}
