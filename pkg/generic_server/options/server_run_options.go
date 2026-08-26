package options

import (
	"strings"
	"time"

	"github.com/ClessLi/component-base/pkg/errors"
	"github.com/ClessLi/component-base/pkg/generic_server/server"

	"github.com/spf13/pflag"
)

// APIServerRunOptions contains the options for running a generic API server.
type APIServerRunOptions[Svc any, Store any] struct {
	Mode            string        `json:"mode" mapstructure:"mode"`
	Healthz         bool          `json:"healthz" mapstructure:"healthz"`
	HealthCheckPath string        `json:"health-check-path" mapstructure:"health-check-path"`
	PingTimeout     time.Duration `json:"ping-timeout" mapstructure:"ping-timeout"`
	ShutdownTimeout time.Duration `json:"shutdown-timeout" mapstructure:"shutdown-timeout"`
	Middlewares     []string      `json:"middlewares" mapstructure:"middlewares"`
}

// AddFlags adds flags to the specified FlagSet.
func (o *APIServerRunOptions[Svc, Store]) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Mode, "server.mode", o.Mode,
		"Server running mode. Valid values: debug, test, prod.")
	fs.BoolVar(&o.Healthz, "server.healthz", o.Healthz,
		"Enable healthz endpoint.")
	fs.StringVar(&o.HealthCheckPath, "server.health-check-path", o.HealthCheckPath,
		"Path for health check endpoint.")
	fs.DurationVar(&o.PingTimeout, "server.ping-timeout", o.PingTimeout,
		"Timeout for health check pings.")
	fs.DurationVar(&o.ShutdownTimeout, "server.shutdown-timeout", o.ShutdownTimeout,
		"Graceful shutdown timeout.")
	fs.StringArrayVar(&o.Middlewares, "server.middlewares", o.Middlewares,
		"List of middlewares to enable.")
}

// Validate validates the run options and returns a list of errors if any.
func (o *APIServerRunOptions[Svc, Store]) Validate() []error {
	errs := make([]error, 0)

	// Validate mode
	validModes := map[string]bool{"debug": true, "test": true, "prod": true}
	if o.Mode != "" && !validModes[o.Mode] {
		errs = append(errs, errors.Errorf(
			"--server.mode %q: must be one of debug, test, or prod",
			o.Mode,
		))
	}

	// Validate health check path
	if o.Healthz && o.HealthCheckPath == "" {
		errs = append(errs, errors.New(
			"--server.health-check-path: must not be empty when healthz is enabled",
		))
	} else if o.HealthCheckPath != "" {
		if !strings.HasPrefix(o.HealthCheckPath, "/") {
			errs = append(errs, errors.Errorf(
				"--server.health-check-path %q: must start with '/'",
				o.HealthCheckPath,
			))
		}
	}

	// Validate ping timeout
	if o.PingTimeout < 0 {
		errs = append(errs, errors.Errorf(
			"--server.ping-timeout %v: must be greater than or equal to 0",
			o.PingTimeout,
		))
	}

	// Validate shutdown timeout
	if o.ShutdownTimeout < 0 {
		errs = append(errs, errors.Errorf(
			"--server.shutdown-timeout %v: must be greater than or equal to 0",
			o.ShutdownTimeout,
		))
	}

	return errs
}

// Complete fills in default values for unset fields.
func (o *APIServerRunOptions[Svc, Store]) Complete() error {
	if o.Mode == "" {
		o.Mode = "debug"
	}
	if o.HealthCheckPath == "" {
		o.HealthCheckPath = "/healthz"
	}
	if o.PingTimeout == 0 {
		o.PingTimeout = 30 * time.Second
	}
	if o.ShutdownTimeout == 0 {
		o.ShutdownTimeout = 30 * time.Second
	}
	if o.Middlewares == nil {
		o.Middlewares = []string{}
	}
	return nil
}

// ApplyTo applies the run options to the server config.
func (o *APIServerRunOptions[Svc, Store]) ApplyTo(c *server.APIServerConfig[Svc, Store]) error {
	c.Mode = o.Mode
	c.Healthz = o.Healthz
	c.HealthCheckPath = o.HealthCheckPath
	c.PingTimeout = o.PingTimeout
	c.ShutdownTimeout = o.ShutdownTimeout
	c.Middlewares = o.Middlewares
	return nil
}

// NewAPIServerRunOptions creates a new APIServerRunOptions with default values.
func NewAPIServerRunOptions[Svc any, Store any]() (o *APIServerRunOptions[Svc, Store]) {
	return &APIServerRunOptions[Svc, Store]{
		Mode:            "debug",
		Healthz:         true,
		HealthCheckPath: "/healthz",
		PingTimeout:     30 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		Middlewares:     []string{},
	}
}

// GRPCServerRunOptions contains the options for running a generic gRPC server.
type GRPCServerRunOptions[Svc any] struct {
	Healthz     bool     `json:"healthz"     mapstructure:"healthz"`
	Middlewares []string `json:"middlewares" mapstructure:"middlewares"`
}

// NewGRPCServerRunOptions creates a new GRPCServerRunOptions with default values.
func NewGRPCServerRunOptions[Svc any]() (o *GRPCServerRunOptions[Svc]) {
	defaults := server.NewGRPCServerConfig[Svc]()

	return &GRPCServerRunOptions[Svc]{
		Healthz:     defaults.Healthz,
		Middlewares: defaults.Middlewares,
	}
}

// AddFlags adds flags for the gRPC server to the specified FlagSet.
func (s *GRPCServerRunOptions[Svc]) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&s.Healthz, "server.healthz", s.Healthz,
		"Enable health check service.")

	fs.StringSliceVar(&s.Middlewares, "server.middlewares", s.Middlewares,
		"List of middlewares to enable. If empty, default middlewares will be used.")
}

// Validate validates the gRPC server run options.
func (s *GRPCServerRunOptions[Svc]) Validate() []error {
	// TODO: implement validation logic
	return nil
}

// Complete fills in default values for unset fields.
func (s *GRPCServerRunOptions[Svc]) Complete() error {
	if s.Middlewares == nil {
		s.Middlewares = []string{}
	}
	return nil
}

// ApplyTo applies the run options to the gRPC server config.
func (s *GRPCServerRunOptions[Svc]) ApplyTo(c *server.GRPCServerConfig[Svc]) error {
	c.Healthz = s.Healthz
	c.Middlewares = s.Middlewares

	return nil
}
