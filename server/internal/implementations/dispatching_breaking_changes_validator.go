package implementations

import (
	"context"
	"fmt"

	"github.com/user/protocol_registry/internal/entities"
)

// DispatchingBreakingChangesValidator routes breaking change detection to the
// appropriate implementation based on the protocol type.
type DispatchingBreakingChangesValidator struct {
	grpcValidator    *BreakingChangesValidatorProtocompile
	openapiValidator *BreakingChangesValidatorOpenAPI
}

func NewDispatchingBreakingChangesValidator(
	grpc *BreakingChangesValidatorProtocompile,
	openapi *BreakingChangesValidatorOpenAPI,
) *DispatchingBreakingChangesValidator {
	return &DispatchingBreakingChangesValidator{
		grpcValidator:    grpc,
		openapiValidator: openapi,
	}
}

func (v *DispatchingBreakingChangesValidator) Validate(ctx context.Context, protocolType entities.ProtocolType, previous, current entities.ProtoFileSet) error {
	switch protocolType {
	case entities.ProtocolTypeGRPC:
		return v.grpcValidator.Validate(ctx, protocolType, previous, current)
	case entities.ProtocolTypeOpenAPI:
		return v.openapiValidator.Validate(ctx, protocolType, previous, current)
	default:
		return fmt.Errorf("unsupported protocol type for breaking change validation: %v", protocolType)
	}
}
