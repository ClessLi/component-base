// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package apiserver

import (
	httpmwv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/http/middleware/v1"
	genericserver "github.com/ClessLi/component-base/pkg/generic_server/server"
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	"github.com/gin-gonic/gin"
)

// bizHooks implements server.APIServerBizHooks and wires the five layers together.
// The generic server calls these hooks in order during PrepareRun/Run/Close.
type bizHooks struct{}

// Compile-time check that bizHooks satisfies the generic server contract.
var _ genericserver.APIServerBizHooks[ServiceFactory, StoreFactory] = (*bizHooks)(nil)

// NewBizHooks creates the business lifecycle hooks for the example server.
func NewBizHooks() *bizHooks {
	return &bizHooks{}
}

// InitStore initializes the store layer.
func (h *bizHooks) InitStore() (StoreFactory, error) {
	logV1.Debug("Initializing store layer")
	return NewStoreFactory(), nil
}

// InitService initializes the service layer with the store instance.
func (h *bizHooks) InitService(store StoreFactory) (ServiceFactory, error) {
	logV1.Debug("Initializing service layer")
	return NewServiceFactory(store), nil
}

// RegisteredMiddlewares returns the middleware registry for the service factory.
func (h *bizHooks) RegisteredMiddlewares() (httpmwv1.Middlewares[ServiceFactory], error) {
	return GetMiddlewares(), nil
}

// InitController registers all routes and controllers with the gin router.
func (h *bizHooks) InitController(router *gin.Engine, svc ServiceFactory) error {
	logV1.Debug("Initializing controller layer")

	// Propagate the real client IP into the request context for methodlog.
	router.Use(clientIPMiddleware())

	ctl := NewUserController(svc.Users())
	v1 := router.Group("/v1")
	{
		v1.GET("/users", ctl.List)
		v1.GET("/users/:id", ctl.Get)
		v1.POST("/users", ctl.Create)
	}
	return nil
}

// Startup starts business resources before the server begins listening.
func (h *bizHooks) Startup() error {
	logV1.Debug("Starting business resources")
	return nil
}

// Shutdown cleans up business resources during graceful shutdown.
func (h *bizHooks) Shutdown() error {
	logV1.Debug("Shutting down business resources")
	return nil
}
