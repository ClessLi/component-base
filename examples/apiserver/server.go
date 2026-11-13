// Copyright 2024 ClessLi. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package apiserver

import (
	"context"

	"github.com/ClessLi/component-base/pkg/errors"
	genericoptions "github.com/ClessLi/component-base/pkg/generic_server/options"
	genericserver "github.com/ClessLi/component-base/pkg/generic_server/server"
)

// Options holds the options for running the example API server. It mirrors the
// generic options used by gjallarhorn, limited to what this example needs.
type Options struct {
	APIServerRunOptions    *genericoptions.APIServerRunOptions[ServiceFactory, StoreFactory]
	InsecureServingOptions *genericoptions.InsecureServingOptions[ServiceFactory, StoreFactory]
}

// NewOptions creates Options with sensible defaults for the example.
func NewOptions() *Options {
	opts := &Options{
		APIServerRunOptions:    genericoptions.NewAPIServerRunOptions[ServiceFactory, StoreFactory](),
		InsecureServingOptions: genericoptions.NewInsecureServingOptions[ServiceFactory, StoreFactory](),
	}
	opts.APIServerRunOptions.Mode = "test"
	opts.APIServerRunOptions.Middlewares = []string{"logging"}
	opts.InsecureServingOptions.BindAddress = "127.0.0.1"
	opts.InsecureServingOptions.BindPort = 8080
	return opts
}

// CreateConfigFromOptions builds an APIServerConfig from Options, mirroring
// gjallarhorn's config.CreateConfigFromOptions: validate, complete, then apply.
func CreateConfigFromOptions(opts *Options) (*genericserver.APIServerConfig[ServiceFactory, StoreFactory], error) {
	if errs := opts.APIServerRunOptions.Validate(); len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}
	if errs := opts.InsecureServingOptions.Validate(); len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}
	if err := opts.APIServerRunOptions.Complete(); err != nil {
		return nil, err
	}
	if err := opts.InsecureServingOptions.Complete(); err != nil {
		return nil, err
	}

	cfg := genericserver.NewAPIServerConfig[ServiceFactory, StoreFactory]()
	if err := opts.APIServerRunOptions.ApplyTo(cfg); err != nil {
		return nil, err
	}
	if err := opts.InsecureServingOptions.ApplyToAPIConfig(cfg); err != nil {
		return nil, err
	}

	// Keep the example lean: disable the metrics and profiling endpoints.
	cfg.EnableMetrics = false
	cfg.EnableProfiling = false
	return cfg, nil
}

// NewServer creates the generic API server from config with the example hooks.
func NewServer(
	ctx context.Context,
	cfg *genericserver.APIServerConfig[ServiceFactory, StoreFactory],
) *genericserver.GenericAPIServer[ServiceFactory, StoreFactory] {
	return cfg.Complete().NewAPIServer(ctx, NewBizHooks())
}

// Run builds the server from options and runs it until a shutdown signal arrives.
// It mirrors gjallarhorn's Run entrypoint: create config, then PrepareRun().Run().
func Run(opts *Options) error {
	cfg, err := CreateConfigFromOptions(opts)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return NewServer(ctx, cfg).PrepareRun().Run()
}
