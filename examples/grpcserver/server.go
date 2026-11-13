// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package grpcserver

import (
	"context"

	"github.com/ClessLi/component-base/pkg/errors"
	genericoptions "github.com/ClessLi/component-base/pkg/generic_server/options"
	genericserver "github.com/ClessLi/component-base/pkg/generic_server/server"
)

// Options holds the configuration options for the example gRPC server.
type Options struct {
	GRPCServerRunOptions   *genericoptions.GRPCServerRunOptions[ServiceFactory, StoreFactory]
	GRPCServingOptions     *genericoptions.GRPCServingOptions[ServiceFactory, StoreFactory]
	InsecureServingOptions *genericoptions.InsecureServingOptions[ServiceFactory, StoreFactory]
}

// NewOptions creates Options with sensible defaults for the example.
func NewOptions() *Options {
	runOpts := genericoptions.NewGRPCServerRunOptions[ServiceFactory, StoreFactory]()
	runOpts.Middlewares = []string{"logging"}

	servingOpts := genericoptions.NewGRPCServingOptions[ServiceFactory, StoreFactory]()

	insecureOpts := genericoptions.NewInsecureServingOptions[ServiceFactory, StoreFactory]()
	insecureOpts.BindAddress = "127.0.0.1"
	insecureOpts.BindPort = 8081

	return &Options{
		GRPCServerRunOptions:   runOpts,
		GRPCServingOptions:     servingOpts,
		InsecureServingOptions: insecureOpts,
	}
}

// CreateConfigFromOptions validates, completes and applies options to a server config.
func CreateConfigFromOptions(opts *Options) (*genericserver.GRPCServerConfig[ServiceFactory, StoreFactory], error) {
	errs := make([]error, 0)
	errs = append(errs, opts.GRPCServerRunOptions.Validate()...)
	errs = append(errs, opts.GRPCServingOptions.Validate()...)
	errs = append(errs, opts.InsecureServingOptions.Validate()...)
	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}

	if err := opts.GRPCServerRunOptions.Complete(); err != nil {
		return nil, err
	}
	if err := opts.GRPCServingOptions.Complete(); err != nil {
		return nil, err
	}
	if err := opts.InsecureServingOptions.Complete(); err != nil {
		return nil, err
	}

	cfg := genericserver.NewGRPCServerConfig[ServiceFactory, StoreFactory]()
	if err := opts.GRPCServerRunOptions.ApplyTo(cfg); err != nil {
		return nil, err
	}
	if err := opts.GRPCServingOptions.ApplyTo(cfg); err != nil {
		return nil, err
	}
	if err := opts.InsecureServingOptions.ApplyToGRPCConfig(cfg); err != nil {
		return nil, err
	}

	// 示例关闭 metrics/profiling，避免 prometheus 全局注册副作用。
	cfg.EnableMetrics = false
	cfg.EnableProfiling = false

	return cfg, nil
}

// NewServer builds a generic gRPC server from the config and business hooks.
func NewServer(ctx context.Context, cfg *genericserver.GRPCServerConfig[ServiceFactory, StoreFactory]) *genericserver.GenericGRPCServer[ServiceFactory, StoreFactory] {
	completed := cfg.Complete()
	return completed.NewGRPCServer(ctx, NewBizHooks(completed.GRPCServing.ChunkSize))
}

// Run wires options -> config -> server and runs until shutdown.
func Run(opts *Options) error {
	cfg, err := CreateConfigFromOptions(opts)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return NewServer(ctx, cfg).PrepareRun().Run()
}
