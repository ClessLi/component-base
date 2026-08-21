// Package v1 provides generic HTTP transport layer utilities.
// It defines Decoder, Encoder, and Handler interfaces with generic type constraints
// for building type-safe HTTP handlers.
package v1

import (
	"net/http"

	epv1 "github.com/ClessLi/component-base/pkg/generic_server/framework/grpc/endpoint/v1"
	"github.com/gin-gonic/gin"
)

// Handler defines the HTTP handler interface with generic type constraints.
// HTTPReq is the HTTP request type, HTTPResp is the HTTP response type.
type Handler[HTTPReq, HTTPResp any] interface {
	ServeHTTP(c *gin.Context)
}

// handler implements the Handler interface with full type constraints.
// HTTPReq: HTTP request type (e.g., JSON body struct)
// DomainReq: domain request type (after decoding)
// DomainResp: domain response type (before encoding)
// HTTPResp: HTTP response type
type handler[HTTPReq, DomainReq, DomainResp, HTTPResp any] struct {
	endpoint epv1.Endpoint[DomainReq, DomainResp]
	decoder  Decoder[HTTPReq, DomainReq]
	encoder  Encoder[DomainResp, HTTPResp]
}

// ServeHTTP handles the HTTP request by decoding, calling the endpoint, and encoding the response.
func (h *handler[HTTPReq, DomainReq, DomainResp, HTTPResp]) ServeHTTP(c *gin.Context) {
	var httpReq HTTPReq

	// Bind HTTP request
	if err := c.ShouldBindJSON(&httpReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Decode HTTP request to domain request
	domainReq, err := h.decoder(c.Request.Context(), httpReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call endpoint
	domainResp, err := h.endpoint(c.Request.Context(), domainReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Encode domain response to HTTP response
	httpResp, err := h.encoder(c.Request.Context(), domainResp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, httpResp)
}

// NewHandler creates a new HTTP handler with the given endpoint, decoder, and encoder.
// The handler provides type-safe conversion between HTTP types and domain types.
func NewHandler[HTTPReq, DomainReq, DomainResp, HTTPResp any](
	ep epv1.Endpoint[DomainReq, DomainResp],
	decoder Decoder[HTTPReq, DomainReq],
	encoder Encoder[DomainResp, HTTPResp],
) Handler[HTTPReq, HTTPResp] {
	return &handler[HTTPReq, DomainReq, DomainResp, HTTPResp]{
		endpoint: ep,
		decoder:  decoder,
		encoder:  encoder,
	}
}
