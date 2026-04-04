package implementations

import (
	"context"

	"github.com/user/protocol-registry-cli/internal/entities"
	"github.com/user/protocol-registry-cli/internal/usecases/get_grpc_view"
	"github.com/user/protocol-registry-cli/internal/usecases/get_protocol"
	"github.com/user/protocol-registry-cli/internal/usecases/publish_protocol"
	"github.com/user/protocol-registry-cli/internal/usecases/register_consumer"
	"github.com/user/protocol-registry-cli/internal/usecases/unregister_consumer"
	"github.com/user/protocol-registry-cli/internal/usecases/validate_protocol"
	registryv1 "github.com/user/protocol_registry/pkg/api/registry/v1"
)

type RegistryClientGRPC struct {
	client registryv1.ProtocolRegistryClient
}

func NewRegistryClientGRPC(client registryv1.ProtocolRegistryClient) *RegistryClientGRPC {
	return &RegistryClientGRPC{client: client}
}

func (r *RegistryClientGRPC) PublishProtocol(ctx context.Context, serviceName string, protocolType entities.ProtocolType, files []publish_protocol.ProtoFile, entryPoint string, versions []string) (*publish_protocol.Output, error) {
	resp, err := r.client.PublishProtocol(ctx, &registryv1.PublishProtocolRequest{
		ServiceName:  serviceName,
		ProtocolType: toProtoType(protocolType),
		Files:        toProtoFiles(files),
		EntryPoint:   entryPoint,
		Versions:     versions,
	})
	if err != nil {
		return nil, err
	}

	return &publish_protocol.Output{
		ServiceName: resp.GetServiceName(),
		IsNew:       resp.GetIsNew(),
	}, nil
}

func (r *RegistryClientGRPC) GetProtocol(ctx context.Context, serviceName string, protocolType entities.ProtocolType, version string) (*get_protocol.Output, error) {
	resp, err := r.client.GetProtocol(ctx, &registryv1.GetProtocolRequest{
		ServiceName:  serviceName,
		ProtocolType: toProtoType(protocolType),
		Version:      version,
	})
	if err != nil {
		return nil, err
	}

	return &get_protocol.Output{
		ServiceName: resp.GetServiceName(),
		EntryPoint:  resp.GetEntryPoint(),
		Files:       fromProtoFiles(resp.GetFiles()),
	}, nil
}

func (r *RegistryClientGRPC) RegisterConsumer(ctx context.Context, consumerName, serverName string, protocolType entities.ProtocolType, files []register_consumer.ProtoFile, entryPoint string, serverVersions []string) (*register_consumer.Output, error) {
	resp, err := r.client.RegisterConsumer(ctx, &registryv1.RegisterConsumerRequest{
		ConsumerName:   consumerName,
		ServerName:     serverName,
		ProtocolType:   toProtoType(protocolType),
		Files:          toRegisterProtoFiles(files),
		EntryPoint:     entryPoint,
		ServerVersions: serverVersions,
	})
	if err != nil {
		return nil, err
	}

	return &register_consumer.Output{
		ConsumerName: resp.GetConsumerName(),
		ServerName:   resp.GetServerName(),
		IsNew:        resp.GetIsNew(),
	}, nil
}

func (r *RegistryClientGRPC) UnregisterConsumer(ctx context.Context, consumerName, serverName string, protocolType entities.ProtocolType, serverVersions []string) error {
	_, err := r.client.UnregisterConsumer(ctx, &registryv1.UnregisterConsumerRequest{
		ConsumerName:   consumerName,
		ServerName:     serverName,
		ProtocolType:   toProtoType(protocolType),
		ServerVersions: serverVersions,
	})
	return err
}

func (r *RegistryClientGRPC) GetGrpcView(ctx context.Context, serviceName string) (*get_grpc_view.Output, error) {
	resp, err := r.client.GetGrpcView(ctx, &registryv1.GetGrpcViewRequest{
		ServiceName: serviceName,
	})
	if err != nil {
		return nil, err
	}

	return fromGrpcViewResponse(resp), nil
}

func (r *RegistryClientGRPC) ValidateProtocol(ctx context.Context, serviceName string, protocolType entities.ProtocolType, files []validate_protocol.ProtoFile, entryPoint string, againstVersions []string) (*validate_protocol.Output, error) {
	protoFiles := make([]*registryv1.ProtoFile, len(files))
	for i, f := range files {
		protoFiles[i] = &registryv1.ProtoFile{
			Path:    f.Path,
			Content: f.Content,
		}
	}

	resp, err := r.client.ValidateProtocol(ctx, &registryv1.ValidateProtocolRequest{
		ServiceName:     serviceName,
		ProtocolType:    toProtoType(protocolType),
		Files:           protoFiles,
		EntryPoint:      entryPoint,
		AgainstVersions: againstVersions,
	})
	if err != nil {
		return nil, err
	}

	violations := make([]validate_protocol.VersionViolation, 0, len(resp.GetVersionViolations()))
	for _, vv := range resp.GetVersionViolations() {
		consumers := make([]validate_protocol.ConsumerViolation, 0, len(vv.GetConsumers()))
		for _, cv := range vv.GetConsumers() {
			consumers = append(consumers, validate_protocol.ConsumerViolation{
				ConsumerName: cv.GetConsumerName(),
				Violations:   cv.GetViolations(),
			})
		}
		violations = append(violations, validate_protocol.VersionViolation{
			Version:   vv.GetVersion(),
			Consumers: consumers,
		})
	}

	return &validate_protocol.Output{
		Valid:      resp.GetValid(),
		Violations: violations,
	}, nil
}

func toProtoType(t entities.ProtocolType) registryv1.ProtocolType {
	switch t {
	case entities.ProtocolTypeGRPC:
		return registryv1.ProtocolType_PROTOCOL_TYPE_GRPC
	default:
		return registryv1.ProtocolType_PROTOCOL_TYPE_UNSPECIFIED
	}
}

func toProtoFiles(files []publish_protocol.ProtoFile) []*registryv1.ProtoFile {
	result := make([]*registryv1.ProtoFile, len(files))
	for i, f := range files {
		result[i] = &registryv1.ProtoFile{
			Path:    f.Path,
			Content: f.Content,
		}
	}
	return result
}

func toRegisterProtoFiles(files []register_consumer.ProtoFile) []*registryv1.ProtoFile {
	result := make([]*registryv1.ProtoFile, len(files))
	for i, f := range files {
		result[i] = &registryv1.ProtoFile{
			Path:    f.Path,
			Content: f.Content,
		}
	}
	return result
}

func fromProtoFiles(files []*registryv1.ProtoFile) []get_protocol.ProtoFile {
	result := make([]get_protocol.ProtoFile, len(files))
	for i, f := range files {
		result[i] = get_protocol.ProtoFile{
			Path:    f.GetPath(),
			Content: f.GetContent(),
		}
	}
	return result
}

func fromGrpcViewResponse(resp *registryv1.GetGrpcViewResponse) *get_grpc_view.Output {
	services := make([]get_grpc_view.ServiceView, 0, len(resp.GetServices()))
	for _, svc := range resp.GetServices() {
		methods := make([]get_grpc_view.MethodView, 0, len(svc.GetMethods()))
		for _, m := range svc.GetMethods() {
			methods = append(methods, get_grpc_view.MethodView{
				Name:      m.GetName(),
				Input:     fromProtoMessageView(m.GetInput()),
				Output:    fromProtoMessageView(m.GetOutput()),
				Consumers: m.GetConsumers(),
			})
		}
		services = append(services, get_grpc_view.ServiceView{
			Name:    svc.GetName(),
			Methods: methods,
		})
	}

	return &get_grpc_view.Output{
		ServiceName: resp.GetServiceName(),
		Services:    services,
	}
}

func fromProtoMessageView(msg *registryv1.GrpcMessageView) get_grpc_view.MessageView {
	if msg == nil {
		return get_grpc_view.MessageView{}
	}
	fields := make([]get_grpc_view.FieldView, 0, len(msg.GetFields()))
	for _, f := range msg.GetFields() {
		fields = append(fields, get_grpc_view.FieldView{
			Name:      f.GetName(),
			Type:      f.GetType(),
			Number:    int32(f.GetNumber()),
			Consumers: f.GetConsumers(),
		})
	}
	return get_grpc_view.MessageView{
		Name:   msg.GetName(),
		Fields: fields,
	}
}

// Verify interface satisfaction at compile time.
var _ publish_protocol.RegistryClient = (*RegistryClientGRPC)(nil)
var _ get_protocol.RegistryClient = (*RegistryClientGRPC)(nil)
var _ register_consumer.RegistryClient = (*RegistryClientGRPC)(nil)
var _ unregister_consumer.RegistryClient = (*RegistryClientGRPC)(nil)
var _ get_grpc_view.RegistryClient = (*RegistryClientGRPC)(nil)
var _ validate_protocol.RegistryClient = (*RegistryClientGRPC)(nil)
