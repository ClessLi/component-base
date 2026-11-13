package options

import (
	"github.com/ClessLi/component-base/pkg/errors"
	"github.com/ClessLi/component-base/pkg/generic_server/server"

	"github.com/spf13/pflag"
)

// RAOptions contains the options used to connect to the RA server.
type RAOptions[Svc any, Store any] struct {
	Host string `json:"host" mapstructure:"host"`
	Port int    `json:"port" mapstructure:"port"`
}

// NewRAOptions creates a new RAOptions object with default parameters.
func NewRAOptions[Svc any, Store any]() *RAOptions[Svc, Store] {
	return &RAOptions[Svc, Store]{}
}

// AddFlags adds command-line flags for RAOptions.
func (ra *RAOptions[Svc, Store]) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&ra.Host, "ra.host", ra.Host,
		"Specifies the bind address of the Registration Authority server. Set empty to disable.")

	fs.IntVar(&ra.Port, "ra.port", ra.Port,
		"Specifies the bind port of the Registration Authority server.")
}

// Validate validates the RA options.
func (ra *RAOptions[Svc, Store]) Validate() []error {
	errs := make([]error, 0)

	if len(ra.Host) > 0 && (ra.Port < 1 || ra.Port > 65535) {
		errs = append(errs, errors.Errorf(
			"--ra.port %v must be between 1 and 65535, inclusive. It cannot be turned off with 0",
			ra.Port,
		))
	}

	return errs
}

// Complete fills in default values for unset fields.
func (ra *RAOptions[Svc, Store]) Complete() error {
	return nil
}

// ApplyTo applies the RA options to the gRPC server config.
func (ra *RAOptions[Svc, Store]) ApplyTo(c *server.GRPCServerConfig[Svc, Store]) error {
	if ra == nil || ra.Port == 0 {
		c.RA = nil
		return nil
	}

	if c.RA == nil {
		c.RA = &server.RAInfo{}
	}

	c.RA.Host = ra.Host
	c.RA.Port = ra.Port

	return nil
}
