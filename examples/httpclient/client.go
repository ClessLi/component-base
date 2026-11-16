// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package httpclient demonstrates how to build a business client SDK on top of
// component-base's HTTP client SDK (pkg/client-sdk/http/v1). It is the client
// counterpart of examples/apiserver and follows the same four-layer layout as
// the eir apiserver client: transport -> endpoint -> service -> client factory.
package httpclient

import (
	"context"
	"net/http"

	epclientv1 "github.com/ClessLi/component-base/examples/httpclient/endpoint"
	svcclientv1 "github.com/ClessLi/component-base/examples/httpclient/service"
	txpclientv1 "github.com/ClessLi/component-base/examples/httpclient/transport"
)

// Client is the example HTTP client. It embeds the service-layer Factory so
// callers use business methods directly (e.g. client.Users().Get("1")).
type Client struct {
	svcclientv1.Factory
}

// NewClient creates a Client for the given base URL. A nil httpClient falls
// back to http.DefaultClient; ctx seeds every request's context.
func NewClient(ctx context.Context, baseURL string, httpClient *http.Client) *Client {
	if ctx == nil {
		ctx = context.Background()
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &Client{
		Factory: svcclientv1.NewService(ctx, epclientv1.NewEndpoint(txpclientv1.NewTransport(httpClient, baseURL))),
	}
}

// Close releases client resources. The shared http.Client needs no explicit
// shutdown here, so it is a no-op kept for API symmetry.
func (c *Client) Close() error {
	return nil
}
