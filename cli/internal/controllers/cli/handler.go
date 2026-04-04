package cli

import (
	"io"

	"github.com/user/protocol-registry-cli/internal/app"
	"github.com/user/protocol-registry-cli/internal/config"
	"github.com/user/protocol-registry-cli/internal/usecases/get_protocol"
	"github.com/user/protocol-registry-cli/internal/usecases/publish_protocol"
	"github.com/user/protocol-registry-cli/internal/usecases/register_consumer"
	"github.com/user/protocol-registry-cli/internal/usecases/unregister_consumer"
	"github.com/user/protocol-registry-cli/internal/usecases/validate_protocol"
	urfave "github.com/urfave/cli/v2"
)

type Handler struct {
	publishUC    *publish_protocol.UseCase
	getUC        *get_protocol.UseCase
	registerUC   *register_consumer.UseCase
	unregisterUC *unregister_consumer.UseCase
	validateUC   *validate_protocol.UseCase
	closer       io.Closer
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Init(addr string) error {
	a, err := app.New(config.Config{ServerAddr: addr})
	if err != nil {
		return err
	}

	h.publishUC = a.PublishUC
	h.getUC = a.GetUC
	h.registerUC = a.RegisterUC
	h.unregisterUC = a.UnregisterUC
	h.validateUC = a.ValidateUC
	h.closer = a

	return nil
}

func (h *Handler) Close() error {
	if h.closer != nil {
		return h.closer.Close()
	}
	return nil
}

func (h *Handler) Commands() []*urfave.Command {
	return []*urfave.Command{
		h.initCommand(),
		h.serverCommand(),
		h.clientCommand(),
	}
}
