// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package apiserver

import (
	"net/http"

	"github.com/ClessLi/component-base/pkg/core"
	methodlog "github.com/ClessLi/component-base/pkg/methodlog/v1"
	"github.com/gin-gonic/gin"
)

// UserController handles HTTP requests for the user resource. It is the
// transport/controller layer: it binds requests, delegates to the service layer,
// and writes responses through core.WriteResponse.
type UserController struct {
	svc UserService
}

// NewUserController creates a UserController backed by the given service.
func NewUserController(svc UserService) *UserController {
	return &UserController{svc: svc}
}

// Get handles GET /v1/users/:id.
func (ctl *UserController) Get(c *gin.Context) {
	u, err := ctl.svc.Get(c.Request.Context(), c.Param("id"))
	core.WriteResponse(c, err, u)
}

// List handles GET /v1/users.
func (ctl *UserController) List(c *gin.Context) {
	users, err := ctl.svc.List(c.Request.Context())
	core.WriteResponse(c, err, users)
}

// Create handles POST /v1/users.
func (ctl *UserController) Create(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := ctl.svc.Create(c.Request.Context(), req.Name)
	core.WriteResponse(c, err, u)
}

// clientIPMiddleware propagates the real client IP into the request context so
// downstream methodlog (HTTPFieldExtractor) can log it. It is a transport-layer
// concern, registered on the gin router before the business routes; controllers
// keep using c.Request.Context() unchanged.
func clientIPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := methodlog.WithClientIP(c.Request.Context(), c.ClientIP())
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
