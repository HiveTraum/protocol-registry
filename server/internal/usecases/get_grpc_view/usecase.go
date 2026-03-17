package get_grpc_view

import (
	"context"
	"fmt"

	"github.com/user/protocol_registry/internal/entities"
)

type consumerInspection struct {
	name     string
	services []entities.ProtoServiceView
}

type UseCase struct {
	serviceRepo     ServiceRepository
	protocolRepo    ProtocolRepository
	storage         Storage
	consumerRepo    ConsumerRepository
	consumerStorage ConsumerStorage
	protoInspector  ProtoInspector
}

func New(
	serviceRepo ServiceRepository,
	protocolRepo ProtocolRepository,
	storage Storage,
	consumerRepo ConsumerRepository,
	consumerStorage ConsumerStorage,
	protoInspector ProtoInspector,
) *UseCase {
	return &UseCase{
		serviceRepo:     serviceRepo,
		protocolRepo:    protocolRepo,
		storage:         storage,
		consumerRepo:    consumerRepo,
		consumerStorage: consumerStorage,
		protoInspector:  protoInspector,
	}
}

func (uc *UseCase) Execute(ctx context.Context, input Input) (*Output, error) {
	svc, err := uc.serviceRepo.GetByName(ctx, input.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("get service: %w", err)
	}
	if svc == nil {
		return nil, entities.NewServiceNotFoundError(input.ServiceName)
	}

	protocol, err := uc.protocolRepo.GetByServiceAndType(ctx, svc.ID, entities.ProtocolTypeGRPC)
	if err != nil {
		return nil, fmt.Errorf("get protocol: %w", err)
	}
	if protocol == nil {
		return nil, entities.NewProtocolNotFoundError(input.ServiceName)
	}

	serverFileSet, err := uc.storage.DownloadFileSet(ctx, input.ServiceName, entities.ProtocolTypeGRPC)
	if err != nil {
		return nil, fmt.Errorf("download server proto: %w", err)
	}

	serverView, err := uc.protoInspector.Inspect(ctx, serverFileSet)
	if err != nil {
		return nil, fmt.Errorf("inspect server proto: %w", err)
	}

	consumers, err := uc.consumerRepo.ListByServerAndType(ctx, svc.ID, entities.ProtocolTypeGRPC)
	if err != nil {
		return nil, fmt.Errorf("list consumers: %w", err)
	}

	var consumerViews []consumerInspection

	for _, consumer := range consumers {
		consumerSvc, err := uc.serviceRepo.GetByID(ctx, consumer.ConsumerServiceID)
		if err != nil {
			return nil, fmt.Errorf("get consumer service: %w", err)
		}

		consumerFileSet, err := uc.consumerStorage.DownloadConsumerFileSet(ctx, consumerSvc.Name, input.ServiceName, entities.ProtocolTypeGRPC)
		if err != nil {
			return nil, fmt.Errorf("download consumer proto for %q: %w", consumerSvc.Name, err)
		}

		cv, err := uc.protoInspector.Inspect(ctx, consumerFileSet)
		if err != nil {
			return nil, fmt.Errorf("inspect consumer proto for %q: %w", consumerSvc.Name, err)
		}

		consumerViews = append(consumerViews, consumerInspection{
			name:     consumerSvc.Name,
			services: cv,
		})
	}

	outputServices := buildOutputServices(serverView, consumerViews)

	return &Output{
		ServiceName: input.ServiceName,
		Services:    outputServices,
	}, nil
}

func buildOutputServices(serverView []entities.ProtoServiceView, consumerViews []consumerInspection) []ServiceView {
	result := make([]ServiceView, 0, len(serverView))

	for _, svc := range serverView {
		methodViews := make([]MethodView, 0, len(svc.Methods))

		for _, method := range svc.Methods {
			var methodConsumers []string
			inputFields := buildFieldViews(method.Input.Fields)
			outputFields := buildFieldViews(method.Output.Fields)

			for _, cv := range consumerViews {
				if consumerHasMethod(cv.services, svc.Name, method.Name) {
					methodConsumers = append(methodConsumers, cv.name)
					markFieldConsumers(inputFields, cv.services, svc.Name, method.Name, true, cv.name)
					markFieldConsumers(outputFields, cv.services, svc.Name, method.Name, false, cv.name)
				}
			}

			methodViews = append(methodViews, MethodView{
				Name: method.Name,
				Input: MessageView{
					Name:   method.Input.Name,
					Fields: inputFields,
				},
				Output: MessageView{
					Name:   method.Output.Name,
					Fields: outputFields,
				},
				Consumers: methodConsumers,
			})
		}

		result = append(result, ServiceView{
			Name:    svc.Name,
			Methods: methodViews,
		})
	}

	return result
}

func buildFieldViews(fields []entities.ProtoFieldView) []FieldView {
	result := make([]FieldView, 0, len(fields))
	for _, f := range fields {
		result = append(result, FieldView{
			Name:   f.Name,
			Type:   f.Type,
			Number: f.Number,
		})
	}
	return result
}

func consumerHasMethod(consumerServices []entities.ProtoServiceView, svcName, methodName string) bool {
	for _, cs := range consumerServices {
		if cs.Name != svcName {
			continue
		}
		for _, m := range cs.Methods {
			if m.Name == methodName {
				return true
			}
		}
	}
	return false
}

func markFieldConsumers(fields []FieldView, consumerServices []entities.ProtoServiceView, svcName, methodName string, isInput bool, consumerName string) {
	consumerMsg := findConsumerMessage(consumerServices, svcName, methodName, isInput)
	if consumerMsg == nil {
		return
	}

	consumerFieldNumbers := make(map[int32]struct{})
	for _, f := range consumerMsg.Fields {
		consumerFieldNumbers[f.Number] = struct{}{}
	}

	for i := range fields {
		if _, ok := consumerFieldNumbers[fields[i].Number]; ok {
			fields[i].Consumers = append(fields[i].Consumers, consumerName)
		}
	}
}

func findConsumerMessage(consumerServices []entities.ProtoServiceView, svcName, methodName string, isInput bool) *entities.ProtoMessageView {
	for _, cs := range consumerServices {
		if cs.Name != svcName {
			continue
		}
		for _, m := range cs.Methods {
			if m.Name != methodName {
				continue
			}
			if isInput {
				return &m.Input
			}
			return &m.Output
		}
	}
	return nil
}
