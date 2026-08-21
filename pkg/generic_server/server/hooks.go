package server

import (
	mwv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/http/middleware/v1"
	"github.com/gin-gonic/gin"
	"github.com/marmotedu/iam/pkg/shutdown"
)

// APIServerInstallHooks defines the installation lifecycle hooks for the API server.
type APIServerInstallHooks[Svc any, Store any] interface {
	// InitStore initializes the storage layer.
	InitStore() (Store, error)

	// InitService initializes the service layer with the store instance.
	InitService(Store) (Svc, error)

	// InitController registers all routes and controllers.
	InitController(router *gin.Engine, svc Svc) error

	// RegisteredMiddlewares returns the middleware registry for the service type.
	RegisteredMiddlewares() (mwv1.Middlewares[Svc], error)

	// ShutdownFunc returns the shutdown callback for graceful termination.
	ShutdownFunc() shutdown.ShutdownFunc

	// TODO: Add uninstall hooks to decouple resource cleanup logic from ShutdownFunc.
	// UninstallController(router *gin.Engine, svc Svc) error
	// UninstallMiddlewares(svc Svc) error
	// UninstallStore(store Store) error
}
