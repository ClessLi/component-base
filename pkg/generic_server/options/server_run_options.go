package options

import (
	"github.com/ClessLi/component-base/pkg/generic_server/server"

	"github.com/spf13/pflag"
)

// APIServerRunOptions contains the options while running a generic API server.
type APIServerRunOptions[Svc any, Store any] struct {
	Mode            string   `json:"mode" mapstructure:"mode"`
	Healthz         bool     `json:"healthz" mapstructure:"healthz"`
	HealthCheckPath string   `json:"health-check-path" mapstructure:"health-check-path"`
	MaxPingCount    int      `json:"max-ping-count" mapstructure:"max-ping-count"`
	Middlewares     []string `json:"middlewares" mapstructure:"middlewares"`
}

func (o *APIServerRunOptions[Svc, Store]) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Mode, "server.mode", o.Mode,
		"Start server in a specific mode, e.g. debug, test, prod.")
	fs.BoolVar(&o.Healthz, "server.healthz", o.Healthz,
		"Start server with healthz endpoint.")
	fs.StringVar(&o.HealthCheckPath, "server.health-check-path", o.HealthCheckPath,
		"Health check path for checking server health.")
	fs.IntVar(&o.MaxPingCount, "server.max-ping-count", o.MaxPingCount,
		"Maximum ping count for health check.")
	fs.StringArrayVar(&o.Middlewares, "server.middlewares", o.Middlewares,
		"Middlewares for the server.")
}

func (o *APIServerRunOptions[Svc, Store]) Validate() []error {
	return nil
}

func (o *APIServerRunOptions[Svc, Store]) ApplyTo(c *server.APIServerConfig[Svc, Store]) error {
	c.Mode = o.Mode
	c.Healthz = o.Healthz
	c.HealthCheckPath = o.HealthCheckPath
	c.MaxPingCount = o.MaxPingCount
	c.Middlewares = o.Middlewares
	return nil
}

// NewAPIServerRunOptions creates a new APIServerRunOptions object with default parameters.
func NewAPIServerRunOptions[Svc any, Store any]() (o *APIServerRunOptions[Svc, Store]) {
	return &APIServerRunOptions[Svc, Store]{
		Mode:            "debug",
		Healthz:         true,
		HealthCheckPath: "/healthz",
		MaxPingCount:    3,
		Middlewares:     []string{},
	}
}

// GRPCServerRunOptions contains the options while running a generic api server.
type GRPCServerRunOptions[Svc any] struct {
	Healthz     bool     `json:"healthz"     mapstructure:"healthz"`
	Middlewares []string `json:"middlewares" mapstructure:"middlewares"`
}

// NewGRPCServerRunOptions creates a new GRPCServerRunOptions object with default parameters.
func NewGRPCServerRunOptions[Svc any](namespace string) (o *GRPCServerRunOptions[Svc]) {
	defaults := server.NewGRPCServerConfig[Svc]()

	return &GRPCServerRunOptions[Svc]{
		Healthz:     defaults.Healthz,
		Middlewares: defaults.Middlewares,
	}
}

// AddFlags adds flags for a specific GRPCServer to the specified FlagSet.
func (s *GRPCServerRunOptions[Svc]) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&s.Healthz, "server.healthz", s.Healthz, ""+
		"Add self readiness check and install health check service.")

	fs.StringSliceVar(&s.Middlewares, "server.middlewares", s.Middlewares, ""+
		"List of allowed middlewares for server, comma separated. If this is empty default middlewares will be used.")
}

// Validate checks validation of GRPCServerRunOptions.
func (s *GRPCServerRunOptions[Svc]) Validate() []error {
	return nil
}

// ApplyTo applies the run options to the method receiver and returns self.
func (s *GRPCServerRunOptions[Svc]) ApplyTo(c *server.GRPCServerConfig[Svc]) error {
	c.Healthz = s.Healthz
	c.Middlewares = s.Middlewares

	return nil
}
