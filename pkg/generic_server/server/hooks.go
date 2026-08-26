package server

import (
	mwv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/http/middleware/v1"
	"github.com/gin-gonic/gin"
)

// APIServerBizHooks defines the business lifecycle hooks for the API server.
type APIServerBizHooks[Svc any, Store any] interface {
	// InitStore initializes the storage layer.
	InitStore() (Store, error)

	// InitService initializes the service layer with the store instance.
	InitService(Store) (Svc, error)

	// InitController registers all routes and controllers with the router.
	InitController(router *gin.Engine, svc Svc) error

	// RegisteredMiddlewares returns the middleware registry for the service type.
	RegisteredMiddlewares() (mwv1.Middlewares[Svc], error)

	// Startup is called after all layers are initialized and before the server starts listening.
	// Use this hook to start business resources such as connection pools, background workers, or schedulers.
	Startup() error

	// Shutdown is called during server graceful shutdown to clean up business resources.
	// Use this hook to stop background workers, close connections, or release resources.
	Shutdown() error

	// TODO: Add uninstall hooks to decouple resource cleanup from Shutdown.
	// UninstallController(router *gin.Engine, svc Svc) error
	// UninstallMiddlewares(svc Svc) error
	// UninstallStore(store Store) error
}
