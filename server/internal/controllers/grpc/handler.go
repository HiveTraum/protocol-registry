package grpc

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/user/protocol_registry/internal/entities"
	"github.com/user/protocol_registry/internal/usecases/get_grpc_view"
	"github.com/user/protocol_registry/internal/usecases/get_protocol"
	"github.com/user/protocol_registry/internal/usecases/list_services"
	"github.com/user/protocol_registry/internal/usecases/publish_protocol"
	"github.com/user/protocol_registry/internal/usecases/register_consumer"
	"github.com/user/protocol_registry/internal/usecases/unregister_consumer"
	"github.com/user/protocol_registry/internal/usecases/validate_protocol"
	registryv1 "github.com/user/protocol_registry/pkg/api/registry/v1"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	registryv1.UnimplementedProtocolRegistryServer
	publishUC      *publish_protocol.UseCase
	getUC          *get_protocol.UseCase
	registerUC     *register_consumer.UseCase
	unregisterUC   *unregister_consumer.UseCase
	grpcViewUC     *get_grpc_view.UseCase
	listServicesUC *list_services.UseCase
	validateUC     *validate_protocol.UseCase
}

func NewHandler(
	publishUC *publish_protocol.UseCase,
	getUC *get_protocol.UseCase,
	registerUC *register_consumer.UseCase,
	unregisterUC *unregister_consumer.UseCase,
	grpcViewUC *get_grpc_view.UseCase,
	listServicesUC *list_services.UseCase,
	validateUC *validate_protocol.UseCase,
) *Handler {
	return &Handler{
		publishUC:      publishUC,
		getUC:          getUC,
		registerUC:     registerUC,
		unregisterUC:   unregisterUC,
		grpcViewUC:     grpcViewUC,
		listServicesUC: listServicesUC,
		validateUC:     validateUC,
	}
}

func (h *Handler) PublishProtocol(ctx context.Context, req *registryv1.PublishProtocolRequest) (*registryv1.PublishProtocolResponse, error) {
	if req.ServiceName == "" {
		return nil, status.Error(codes.InvalidArgument, "service_name is required")
	}
	if req.ProtocolType == registryv1.ProtocolType_PROTOCOL_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "protocol_type is required")
	}
	if err := validateFileSetRequest(req.Files, req.EntryPoint); err != nil {
		return nil, err
	}

	versions := req.Versions
	if len(versions) == 0 {
		versions = []string{"default"}
	}

	input := publish_protocol.Input{
		ServiceName:  req.ServiceName,
		ProtocolType: protoTypeToEntity(req.ProtocolType),
		FileSet:      protoFilesToFileSet(req.Files, req.EntryPoint),
		Versions:     versions,
	}

	output, err := h.publishUC.Execute(ctx, input)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &registryv1.PublishProtocolResponse{
		ServiceName: output.ServiceName,
		IsNew:       output.IsNew,
	}, nil
}

func (h *Handler) GetProtocol(ctx context.Context, req *registryv1.GetProtocolRequest) (*registryv1.GetProtocolResponse, error) {
	if req.ServiceName == "" {
		return nil, status.Error(codes.InvalidArgument, "service_name is required")
	}
	if req.ProtocolType == registryv1.ProtocolType_PROTOCOL_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "protocol_type is required")
	}

	version := req.Version
	if version == "" {
		version = "default"
	}

	input := get_protocol.Input{
		ServiceName:  req.ServiceName,
		ProtocolType: protoTypeToEntity(req.ProtocolType),
		Version:      version,
	}

	output, err := h.getUC.Execute(ctx, input)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &registryv1.GetProtocolResponse{
		ServiceName:  output.ServiceName,
		ProtocolType: req.ProtocolType,
		Files:        fileSetToProtoFiles(output.FileSet),
		EntryPoint:   output.FileSet.EntryPoint,
	}, nil
}

func (h *Handler) RegisterConsumer(ctx context.Context, req *registryv1.RegisterConsumerRequest) (*registryv1.RegisterConsumerResponse, error) {
	if req.ConsumerName == "" {
		return nil, status.Error(codes.InvalidArgument, "consumer_name is required")
	}
	if req.ServerName == "" {
		return nil, status.Error(codes.InvalidArgument, "server_name is required")
	}
	if req.ProtocolType == registryv1.ProtocolType_PROTOCOL_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "protocol_type is required")
	}
	if err := validateFileSetRequest(req.Files, req.EntryPoint); err != nil {
		return nil, err
	}

	serverVersions := req.ServerVersions
	if len(serverVersions) == 0 {
		serverVersions = []string{"default"}
	}

	input := register_consumer.Input{
		ConsumerName:   req.ConsumerName,
		ServerName:     req.ServerName,
		ProtocolType:   protoTypeToEntity(req.ProtocolType),
		FileSet:        protoFilesToFileSet(req.Files, req.EntryPoint),
		ServerVersions: serverVersions,
	}

	output, err := h.registerUC.Execute(ctx, input)
	if err != nil {
		return nil, mapDomainError(err)
	}

	return &registryv1.RegisterConsumerResponse{
		ConsumerName: output.ConsumerName,
		ServerName:   output.ServerName,
		IsNew:        output.IsNew,
	}, nil
}

func (h *Handler) UnregisterConsumer(ctx context.Context, req *registryv1.UnregisterConsumerRequest) (*registryv1.UnregisterConsumerResponse, error) {
	if req.ConsumerName == "" {
		return nil, status.Error(codes.InvalidArgument, "consumer_name is required")
	}
	if req.ServerName == "" {
		return nil, status.Error(codes.InvalidArgument, "server_name is required")
	}
	if req.ProtocolType == registryv1.ProtocolType_PROTOCOL_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "protocol_type is required")
	}

	serverVersions := req.ServerVersions
	if len(serverVersions) == 0 {
		serverVersions = []string{"default"}
	}

	input := unregister_consumer.Input{
		ConsumerName:   req.ConsumerName,
		ServerName:     req.ServerName,
		ProtocolType:   protoTypeToEntity(req.ProtocolType),
		ServerVersions: serverVersions,
	}

	if err := h.unregisterUC.Execute(ctx, input); err != nil {
		return nil, mapDomainError(err)
	}

	return &registryv1.UnregisterConsumerResponse{}, nil
}

func (h *Handler) GetGrpcView(ctx context.Context, req *registryv1.GetGrpcViewRequest) (*registryv1.GetGrpcViewResponse, error) {
	if req.ServiceName == "" {
		return nil, status.Error(codes.InvalidArgument, "service_name is required")
	}

	input := get_grpc_view.Input{
		ServiceName: req.ServiceName,
	}

	output, err := h.grpcViewUC.Execute(ctx, input)
	if err != nil {
		return nil, mapDomainError(err)
	}

	protoServices := make([]*registryv1.GrpcServiceView, 0, len(output.Services))
	for _, svc := range output.Services {
		protoMethods := make([]*registryv1.GrpcMethodView, 0, len(svc.Methods))
		for _, m := range svc.Methods {
			protoMethods = append(protoMethods, &registryv1.GrpcMethodView{
				Name:      m.Name,
				Input:     toProtoMessageView(m.Input),
				Output:    toProtoMessageView(m.Output),
				Consumers: m.Consumers,
			})
		}
		protoServices = append(protoServices, &registryv1.GrpcServiceView{
			Name:    svc.Name,
			Methods: protoMethods,
		})
	}

	return &registryv1.GetGrpcViewResponse{
		ServiceName: output.ServiceName,
		Services:    protoServices,
	}, nil
}

func (h *Handler) ListServices(ctx context.Context, _ *registryv1.ListServicesRequest) (*registryv1.ListServicesResponse, error) {
	output, err := h.listServicesUC.Execute(ctx)
	if err != nil {
		return nil, mapDomainError(err)
	}

	services := make([]*registryv1.ServiceInfo, len(output.Services))
	for i, s := range output.Services {
		services[i] = &registryv1.ServiceInfo{
			Id:   s.ID,
			Name: s.Name,
		}
	}

	return &registryv1.ListServicesResponse{
		Services: services,
	}, nil
}

func (h *Handler) ValidateProtocol(ctx context.Context, req *registryv1.ValidateProtocolRequest) (*registryv1.ValidateProtocolResponse, error) {
	if req.ServiceName == "" {
		return nil, status.Error(codes.InvalidArgument, "service_name is required")
	}
	if req.ProtocolType == registryv1.ProtocolType_PROTOCOL_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "protocol_type is required")
	}
	if err := validateFileSetRequest(req.Files, req.EntryPoint); err != nil {
		return nil, err
	}

	againstVersions := req.AgainstVersions
	if len(againstVersions) == 0 {
		againstVersions = []string{"default"}
	}

	input := validate_protocol.Input{
		ServiceName:     req.ServiceName,
		ProtocolType:    protoTypeToEntity(req.ProtocolType),
		FileSet:         protoFilesToFileSet(req.Files, req.EntryPoint),
		AgainstVersions: againstVersions,
	}

	output, err := h.validateUC.Execute(ctx, input)
	if err != nil {
		return nil, mapDomainError(err)
	}

	protoViolations := make([]*registryv1.VersionViolation, 0, len(output.Violations))
	for _, vv := range output.Violations {
		consumerViolations := make([]*registryv1.ConsumerViolation, 0, len(vv.Consumers))
		for _, cv := range vv.Consumers {
			var msgs []string
			for _, c := range cv.Changes {
				msgs = append(msgs, fmt.Sprintf("[%s] %s", c.Type, c.Message))
			}
			consumerViolations = append(consumerViolations, &registryv1.ConsumerViolation{
				ConsumerName: cv.ConsumerName,
				Violations:   msgs,
			})
		}
		protoViolations = append(protoViolations, &registryv1.VersionViolation{
			Version:   vv.Version,
			Consumers: consumerViolations,
		})
	}

	return &registryv1.ValidateProtocolResponse{
		Valid:             output.Valid,
		VersionViolations: protoViolations,
	}, nil
}

func toProtoMessageView(msg get_grpc_view.MessageView) *registryv1.GrpcMessageView {
	fields := make([]*registryv1.GrpcFieldView, 0, len(msg.Fields))
	for _, f := range msg.Fields {
		fields = append(fields, &registryv1.GrpcFieldView{
			Name:      f.Name,
			Type:      f.Type,
			Number:    uint32(f.Number),
			Consumers: f.Consumers,
		})
	}
	return &registryv1.GrpcMessageView{
		Name:   msg.Name,
		Fields: fields,
	}
}

func protoFilesToFileSet(files []*registryv1.ProtoFile, entryPoint string) entities.ProtoFileSet {
	result := make([]entities.ProtoFile, len(files))
	for i, f := range files {
		result[i] = entities.ProtoFile{
			Path:    f.Path,
			Content: f.Content,
		}
	}
	return entities.ProtoFileSet{
		EntryPoint: entryPoint,
		Files:      result,
	}
}

func fileSetToProtoFiles(fileSet entities.ProtoFileSet) []*registryv1.ProtoFile {
	result := make([]*registryv1.ProtoFile, len(fileSet.Files))
	for i, f := range fileSet.Files {
		result[i] = &registryv1.ProtoFile{
			Path:    f.Path,
			Content: f.Content,
		}
	}
	return result
}

func validateFileSetRequest(files []*registryv1.ProtoFile, entryPoint string) error {
	if len(files) == 0 {
		return status.Error(codes.InvalidArgument, "files is required")
	}
	if entryPoint == "" {
		return status.Error(codes.InvalidArgument, "entry_point is required")
	}

	entryFound := false
	for _, f := range files {
		if f.Path == "" {
			return status.Error(codes.InvalidArgument, "file path must not be empty")
		}
		if path.IsAbs(f.Path) {
			return status.Errorf(codes.InvalidArgument, "file path %q must be relative", f.Path)
		}
		if strings.Contains(f.Path, "..") {
			return status.Errorf(codes.InvalidArgument, "file path %q must not contain '..'", f.Path)
		}
		if f.Path == entryPoint {
			entryFound = true
		}
	}

	if !entryFound {
		return status.Errorf(codes.InvalidArgument, "entry_point %q not found in files", entryPoint)
	}

	return nil
}

func mapDomainError(err error) error {
	var domainErr *entities.DomainError
	if !errors.As(err, &domainErr) {
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}

	switch domainErr.Code {
	case entities.ErrorCodeBreakingChanges:
		return mapBreakingChangesError(domainErr)

	case entities.ErrorCodeConsumerBreakingChanges:
		return mapConsumerBreakingChangesError(domainErr)

	case entities.ErrorCodeSyntaxError:
		return mapSyntaxError(domainErr)

	case entities.ErrorCodeServiceNotFound, entities.ErrorCodeProtocolNotFound:
		return status.Error(codes.NotFound, domainErr.Message)

	case entities.ErrorCodeConsumerNotFound:
		return status.Error(codes.NotFound, domainErr.Message)

	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}

func mapBreakingChangesError(domainErr *entities.DomainError) error {
	st := status.New(codes.FailedPrecondition, "breaking changes detected")

	changes := domainErr.BreakingChanges()
	if len(changes) == 0 {
		return st.Err()
	}

	violations := make([]*errdetails.BadRequest_FieldViolation, 0, len(changes))
	for _, change := range changes {
		violations = append(violations, &errdetails.BadRequest_FieldViolation{
			Field:       change.Subject,
			Description: change.Message,
		})
	}

	detailed, err := st.WithDetails(&errdetails.BadRequest{
		FieldViolations: violations,
	})
	if err != nil {
		return st.Err()
	}

	return detailed.Err()
}

func mapConsumerBreakingChangesError(domainErr *entities.DomainError) error {
	st := status.New(codes.FailedPrecondition, "breaking changes affect consumers")

	consumerChanges := domainErr.ConsumerBreakingChanges()
	if len(consumerChanges) == 0 {
		return st.Err()
	}

	var violations []*errdetails.BadRequest_FieldViolation
	for _, cc := range consumerChanges {
		for _, change := range cc.Changes {
			violations = append(violations, &errdetails.BadRequest_FieldViolation{
				Field:       fmt.Sprintf("consumer:%s/%s", cc.ConsumerName, change.Subject),
				Description: change.Message,
			})
		}
	}

	detailed, err := st.WithDetails(&errdetails.BadRequest{
		FieldViolations: violations,
	})
	if err != nil {
		return st.Err()
	}

	return detailed.Err()
}

func mapSyntaxError(domainErr *entities.DomainError) error {
	st := status.New(codes.InvalidArgument, "proto syntax error")

	syntaxErrs := domainErr.SyntaxErrors()
	if len(syntaxErrs) == 0 {
		return st.Err()
	}

	violations := make([]*errdetails.BadRequest_FieldViolation, 0, len(syntaxErrs))
	for _, e := range syntaxErrs {
		violations = append(violations, &errdetails.BadRequest_FieldViolation{
			Field:       "proto_content",
			Description: e,
		})
	}

	detailed, err := st.WithDetails(&errdetails.BadRequest{
		FieldViolations: violations,
	})
	if err != nil {
		return st.Err()
	}

	return detailed.Err()
}

func protoTypeToEntity(pt registryv1.ProtocolType) entities.ProtocolType {
	switch pt {
	case registryv1.ProtocolType_PROTOCOL_TYPE_GRPC:
		return entities.ProtocolTypeGRPC
	default:
		return entities.ProtocolTypeUnspecified
	}
}
