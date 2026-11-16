// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config is the configuration for the example gRPC client. It mirrors the
// task_scheduler client SDK's config-driven construction, letting callers fill
// in the server address and dial options and then obtain a Client from a
// CompletedConfig.
type Config struct {
	// Address is the target of the example gRPC server (host:port).
	Address string

	// DialOptions are passed to grpc.NewClient when dialing Address.
	DialOptions []grpc.DialOption
}

// NewConfig returns a Config defaulted to the example server's insecure address.
func NewConfig() *Config {
	return &Config{
		Address:     "127.0.0.1:8081",
		DialOptions: []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	}
}

// CompletedConfig is a Config that has had its defaults applied.
type CompletedConfig struct {
	*Config
}

// Complete fills in unset fields with defaults and returns a CompletedConfig.
func (c *Config) Complete() CompletedConfig {
	if c.Address == "" {
		c.Address = "127.0.0.1:8081"
	}
	if len(c.DialOptions) == 0 {
		c.DialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}

	return CompletedConfig{c}
}

// NewClient dials the configured address and returns a Client. The returned
// client owns the connection; call Close when finished.
func (c CompletedConfig) NewClient() (*Client, error) {
	conn, err := grpc.NewClient(c.Address, c.DialOptions...)
	if err != nil {
		return nil, err
	}

	return newClient(context.Background(), conn), nil
}

// NewClientWithContext is NewClient with an explicit root context for requests.
func (c CompletedConfig) NewClientWithContext(ctx context.Context) (*Client, error) {
	conn, err := grpc.NewClient(c.Address, c.DialOptions...)
	if err != nil {
		return nil, err
	}

	return newClient(ctx, conn), nil
}
