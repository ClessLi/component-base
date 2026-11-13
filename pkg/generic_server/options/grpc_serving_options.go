package options

import (
	"github.com/ClessLi/component-base/pkg/errors"
	"github.com/ClessLi/component-base/pkg/generic_server/server"

	"github.com/spf13/pflag"
)

// GRPCServingOptions contains configuration for gRPC serving.
type GRPCServingOptions[Svc any, Store any] struct {
	ChunkSize             int `json:"chunksize"               mapstructure:"chunksize"`
	ReceiveTimeoutMinutes int `json:"receive-timeout-minutes" mapstructure:"receive-timeout-minutes"`
}

// NewGRPCServingOptions creates a new GRPCServingOptions object with default parameters.
func NewGRPCServingOptions[Svc any, Store any]() *GRPCServingOptions[Svc, Store] {
	defaults := server.NewGRPCServingInfo()

	return &GRPCServingOptions[Svc, Store]{
		ChunkSize:             defaults.ChunkSize,
		ReceiveTimeoutMinutes: defaults.RecvTimeoutM,
	}
}

// AddFlags adds command-line flags for GRPCServingOptions.
func (g *GRPCServingOptions[Svc, Store]) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&g.ChunkSize, "grpc.chunksize", g.ChunkSize,
		"Set the max message size in bytes the server can send. Can not less than 100 bytes.")

	fs.IntVar(&g.ReceiveTimeoutMinutes, "grpc.receive-timeout", g.ReceiveTimeoutMinutes,
		"Set the timeout for receiving data. The unit is per minute.")
}

// Validate validates the gRPC serving options.
func (g *GRPCServingOptions[Svc, Store]) Validate() []error {
	errs := make([]error, 0)

	if g.ChunkSize < 100 {
		errs = append(errs, errors.Errorf("--grpc.chunksize %d must great equal 100 bytes.", g.ChunkSize))
	}

	if g.ReceiveTimeoutMinutes <= 0 {
		errs = append(errs, errors.Errorf("--grpc.receive-timeout %d must great than 0.", g.ReceiveTimeoutMinutes))
	}

	return errs
}

// Complete fills in default values for unset fields.
func (g *GRPCServingOptions[Svc, Store]) Complete() error {
	defaults := server.NewGRPCServingInfo()
	if g.ChunkSize == 0 {
		g.ChunkSize = defaults.ChunkSize
	}
	if g.ReceiveTimeoutMinutes == 0 {
		g.ReceiveTimeoutMinutes = defaults.RecvTimeoutM
	}
	return nil
}

// ApplyTo applies the gRPC serving options to the gRPC server config.
func (g *GRPCServingOptions[Svc, Store]) ApplyTo(c *server.GRPCServerConfig[Svc, Store]) error {
	c.GRPCServing = &server.GRPCServingInfo{
		ChunkSize:    g.ChunkSize,
		RecvTimeoutM: g.ReceiveTimeoutMinutes,
	}

	return nil
}
