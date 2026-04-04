package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/user/protocol_registry/internal/usecases/get_grpc_view"
	"github.com/user/protocol_registry/internal/usecases/list_services"
)

type Handler struct {
	listServicesUC *list_services.UseCase
	grpcViewUC     *get_grpc_view.UseCase
}

func NewHandler(
	listServicesUC *list_services.UseCase,
	grpcViewUC *get_grpc_view.UseCase,
) *Handler {
	return &Handler{
		listServicesUC: listServicesUC,
		grpcViewUC:     grpcViewUC,
	}
}

// --- DTOs ---

type ListServicesResponse struct {
	Body struct {
		Services []ServiceDTO `json:"services"`
	}
}

type ServiceDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type GetGrpcViewRequest struct {
	ServiceName string `path:"name" doc:"Service name"`
}

type GetGrpcViewResponse struct {
	Body GrpcViewBody
}

type GrpcViewBody struct {
	ServiceName string           `json:"serviceName"`
	Services    []GrpcServiceDTO `json:"services"`
}

type GrpcServiceDTO struct {
	Name    string          `json:"name"`
	Methods []GrpcMethodDTO `json:"methods"`
}

type GrpcMethodDTO struct {
	Name      string          `json:"name"`
	Input     GrpcMessageDTO  `json:"input"`
	Output    GrpcMessageDTO  `json:"output"`
	Consumers []string        `json:"consumers"`
}

type GrpcMessageDTO struct {
	Name   string         `json:"name"`
	Fields []GrpcFieldDTO `json:"fields"`
}

type GrpcFieldDTO struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Number    int32    `json:"number"`
	Consumers []string `json:"consumers"`
}

// --- Register routes ---

func (h *Handler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-services",
		Method:      http.MethodGet,
		Path:        "/api/services",
		Summary:     "List all registered services",
		Tags:        []string{"services"},
	}, h.listServices)

	huma.Register(api, huma.Operation{
		OperationID: "get-grpc-view",
		Method:      http.MethodGet,
		Path:        "/api/services/{name}/grpc-view",
		Summary:     "Get gRPC view for a service",
		Tags:        []string{"services"},
	}, h.getGrpcView)
}

func (h *Handler) listServices(ctx context.Context, _ *struct{}) (*ListServicesResponse, error) {
	output, err := h.listServicesUC.Execute(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list services", err)
	}

	resp := &ListServicesResponse{}
	resp.Body.Services = make([]ServiceDTO, len(output.Services))
	for i, s := range output.Services {
		resp.Body.Services[i] = ServiceDTO{
			ID:   s.ID,
			Name: s.Name,
		}
	}
	return resp, nil
}

func (h *Handler) getGrpcView(ctx context.Context, input *GetGrpcViewRequest) (*GetGrpcViewResponse, error) {
	output, err := h.grpcViewUC.Execute(ctx, get_grpc_view.Input{
		ServiceName: input.ServiceName,
	})
	if err != nil {
		return nil, huma.Error404NotFound(fmt.Sprintf("service %q not found", input.ServiceName))
	}

	services := make([]GrpcServiceDTO, len(output.Services))
	for i, svc := range output.Services {
		methods := make([]GrpcMethodDTO, len(svc.Methods))
		for j, m := range svc.Methods {
			methods[j] = GrpcMethodDTO{
				Name:      m.Name,
				Input:     toMessageDTO(m.Input),
				Output:    toMessageDTO(m.Output),
				Consumers: m.Consumers,
			}
		}
		services[i] = GrpcServiceDTO{
			Name:    svc.Name,
			Methods: methods,
		}
	}

	resp := &GetGrpcViewResponse{}
	resp.Body = GrpcViewBody{
		ServiceName: output.ServiceName,
		Services:    services,
	}
	return resp, nil
}

func toMessageDTO(msg get_grpc_view.MessageView) GrpcMessageDTO {
	fields := make([]GrpcFieldDTO, len(msg.Fields))
	for i, f := range msg.Fields {
		fields[i] = GrpcFieldDTO{
			Name:      f.Name,
			Type:      f.Type,
			Number:    f.Number,
			Consumers: f.Consumers,
		}
	}
	return GrpcMessageDTO{
		Name:   msg.Name,
		Fields: fields,
	}
}

func NewHTTPServer(h *Handler) http.Handler {
	router := http.NewServeMux()
	api := humago.New(router, huma.DefaultConfig("Protocol Registry API", "1.0.0"))
	h.Register(api)
	return corsMiddleware(router)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
