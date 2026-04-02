package implementations

import (
	"fmt"

	"github.com/user/protocol_registry/internal/entities"
)

// DispatchingSyntaxValidator routes validation to the appropriate implementation
// based on the protocol type.
type DispatchingSyntaxValidator struct {
	grpcValidator    *ProtocolSyntaxValidatorProtocompile
	openapiValidator *OpenAPISyntaxValidator
}

func NewDispatchingSyntaxValidator(
	grpc *ProtocolSyntaxValidatorProtocompile,
	openapi *OpenAPISyntaxValidator,
) *DispatchingSyntaxValidator {
	return &DispatchingSyntaxValidator{
		grpcValidator:    grpc,
		openapiValidator: openapi,
	}
}

func (v *DispatchingSyntaxValidator) Validate(protocolType entities.ProtocolType, fileSet entities.ProtoFileSet) error {
	switch protocolType {
	case entities.ProtocolTypeGRPC:
		return v.grpcValidator.Validate(protocolType, fileSet)
	case entities.ProtocolTypeOpenAPI:
		return v.openapiValidator.Validate(protocolType, fileSet)
	default:
		return entities.NewSyntaxError([]string{fmt.Sprintf("unsupported protocol type: %v", protocolType)})
	}
}
