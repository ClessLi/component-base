package options

import (
	"fmt"

	"github.com/ClessLi/component-base/pkg/errors"
	"github.com/ClessLi/component-base/pkg/generic_server/server"
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"

	"github.com/spf13/pflag"
)

// InsecureServingOptions contains configuration for serving HTTP requests.
type InsecureServingOptions[Svc any, Store any] struct {
	BindAddress string `json:"bind-address" mapstructure:"bind-address"`
	BindPort    int    `json:"bind-port" mapstructure:"bind-port"`
}

// NewInsecureServingOptions creates a new InsecureServingOptions object with default parameters.
func NewInsecureServingOptions[Svc any, Store any]() *InsecureServingOptions[Svc, Store] {
	return &InsecureServingOptions[Svc, Store]{
		BindAddress: "0.0.0.0",
		BindPort:    8080,
	}
}

// Addr returns the address for insecure serving.
func (i *InsecureServingOptions[Svc, Store]) Addr() string {
	if i == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", i.BindAddress, i.BindPort)
}

// ApplyToAPIConfig applies the insecure serving options to the API server config.
func (i *InsecureServingOptions[Svc, Store]) ApplyToAPIConfig(c *server.APIServerConfig[Svc, Store]) error {
	c.InsecureServing = i.buildInsecureServingInfo()

	return nil
}

// ApplyToGRPCConfig applies the insecure serving options to the gRPC server config.
func (i *InsecureServingOptions[Svc, Store]) ApplyToGRPCConfig(c *server.GRPCServerConfig[Svc]) error {
	c.InsecureServing = i.buildInsecureServingInfo()

	return nil
}

// buildInsecureServingInfo builds InsecureServingInfo from options, returns nil if insecure serving is disabled.
func (i *InsecureServingOptions[Svc, Store]) buildInsecureServingInfo() *server.InsecureServingInfo {
	if i == nil || i.BindPort == 0 {
		logV1.Debug("insecure serving is disabled.")
		return nil
	}

	return &server.InsecureServingInfo{
		BindAddress: i.BindAddress,
		BindPort:    i.BindPort,
	}
}

// Validate is used to parse and validate the parameters entered by the user at
// the command line when the program starts.
func (i *InsecureServingOptions[Svc, Store]) Validate() []error {
	if i == nil || i.BindPort == 0 {
		return nil
	}

	errs := make([]error, 0)

	// Validate BindPort range
	if i.BindPort < 0 || i.BindPort > 65535 {
		errs = append(
			errs,
			errors.Errorf(
				"--insecure.bind-port %v: must be between 1 and 65535, inclusive. 0 means disabled",
				i.BindPort,
			),
		)
	}

	return errs
}

// AddFlags adds command-line flags for InsecureServingOptions.
func (i *InsecureServingOptions[Svc, Store]) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&i.BindAddress, "insecure.bind-address", i.BindAddress,
		"The IP address on which to serve the --insecure.bind-port "+
			"(set to 0.0.0.0 for all IPv4 interfaces and :: for all IPv6 interfaces).")

	fs.IntVar(&i.BindPort, "insecure.bind-port", i.BindPort,
		"The port on which to serve unsecured, unauthenticated access. It is assumed "+
			"that firewall rules are set up such that this port is not reachable from outside of "+
			"the deployed machine and that port 443 on the public address is used for "+
			"securing access to the cluster (set to 0 to disable insecure serving).")
}

func (i *InsecureServingOptions[Svc, Store]) Complete() error {
	if i == nil || i.BindPort == 0 {
		return nil
	}

	// Fill default values if not explicitly set
	if i.BindAddress == "" {
		i.BindAddress = "0.0.0.0"
	}

	return nil
}
