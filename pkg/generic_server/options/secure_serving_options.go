package options

import (
	"path/filepath"

	"github.com/ClessLi/component-base/pkg/errors"
	"github.com/ClessLi/component-base/pkg/generic_server/server"
	logV1 "github.com/ClessLi/component-base/pkg/log/v1"

	"github.com/spf13/pflag"
)

// SecureServingOptions contains configuration items related to TLS server startup.
type SecureServingOptions[Svc any, Store any] struct {
	BindAddress string `json:"bind-address" mapstructure:"bind-address"`
	// BindPort is ignored when Listener is set, will disable even with 0.
	BindPort int `json:"bind-port"    mapstructure:"bind-port"`
	// ServerCert is the TLS cert info for serving secure traffic
	ServerCert GeneratableKeyCert `json:"tls"          mapstructure:"tls"`
	// AdvertiseAddress net.IP
}

// CertKey contains configuration items related to certificate.
type CertKey struct {
	// CertFile is a file containing a PEM-encoded certificate, and possibly the complete certificate chain
	CertFile string `json:"cert-file"        mapstructure:"cert-file"`
	// KeyFile is a file containing a PEM-encoded private key for the certificate specified by CertFile
	KeyFile string `json:"private-key-file" mapstructure:"private-key-file"`
}

// GeneratableKeyCert contains configuration items related to certificate.
type GeneratableKeyCert struct {
	// CertKey allows setting an explicit cert/key file to use.
	CertKey CertKey `json:"cert-key" mapstructure:"cert-key"`

	// CertDirectory specifies a directory to write generated certificates to if CertFile/KeyFile aren't explicitly set.
	// PairName is used to determine the filenames within CertDirectory.
	// If CertDirectory and PairName are not set, an in-memory certificate will be generated.
	CertDirectory string `json:"cert-dir"  mapstructure:"cert-dir"`
	// PairName is the name which will be used with CertDirectory to make a cert and key filenames.
	// It becomes CertDirectory/PairName.crt and CertDirectory/PairName.key
	PairName string `json:"pair-name" mapstructure:"pair-name"`
}

// NewSecureServingOptions creates a new SecureServingOptions object with default parameters.
func NewSecureServingOptions[Svc any, Store any]() *SecureServingOptions[Svc, Store] {
	return &SecureServingOptions[Svc, Store]{
		BindAddress: "0.0.0.0",
		BindPort:    0, // default disable secure serving
		ServerCert: GeneratableKeyCert{
			PairName:      "eir-apiserver",
			CertDirectory: "/var/run/eir/apiserver",
		},
	}
}

// ApplyToAPIConfig applies the secure serving options to the API server config.
func (s *SecureServingOptions[Svc, Store]) ApplyToAPIConfig(c *server.APIServerConfig[Svc, Store]) error {
	c.SecureServing = s.buildSecureServingInfo()

	return nil
}

// ApplyToGRPCConfig applies the secure serving options to the gRPC server config.
func (s *SecureServingOptions[Svc, Store]) ApplyToGRPCConfig(c *server.GRPCServerConfig[Svc]) error {
	c.SecureServing = s.buildSecureServingInfo()

	return nil
}

// buildSecureServingInfo builds SecureServingInfo from options, returns nil if secure serving is disabled.
func (s *SecureServingOptions[Svc, Store]) buildSecureServingInfo() *server.SecureServingInfo {
	if s == nil || s.BindPort == 0 {
		logV1.Debug("secure serving is disabled.")
		return nil
	}

	return &server.SecureServingInfo{
		BindAddress: s.BindAddress,
		BindPort:    s.BindPort,
		CertKey: server.CertKey{
			CertFile: s.ServerCert.CertKey.CertFile,
			KeyFile:  s.ServerCert.CertKey.KeyFile,
		},
	}
}

// Validate is used to parse and validate the parameters entered by the user at
// the command line when the program starts.
func (s *SecureServingOptions[Svc, Store]) Validate() []error {
	if s == nil || s.BindPort == 0 {
		return nil
	}

	errs := make([]error, 0)

	// Validate BindPort range
	if s.BindPort < 0 || s.BindPort > 65535 {
		errs = append(
			errs,
			errors.Errorf(
				"--secure.bind-port %v: must be between 1 and 65535, inclusive. 0 means disabled",
				s.BindPort,
			),
		)
	}

	// Validate certificate configuration when secure serving is enabled
	// If both cert and key files are not explicitly specified, validate CertDirectory and PairName
	if s.ServerCert.CertKey.CertFile == "" && s.ServerCert.CertKey.KeyFile == "" {
		if s.ServerCert.CertDirectory == "" {
			errs = append(
				errs,
				errors.Errorf("--secure.tls.cert-dir: required flag is not set when cert-file and private-key-file are not provided"),
			)
		}
		if s.ServerCert.PairName == "" {
			errs = append(
				errs,
				errors.Errorf("--secure.tls.pair-name: required flag is not set when cert-file and private-key-file are not provided"),
			)
		}
	}

	return errs
}

// AddFlags adds command-line flags for SecureServingOptions.
func (s *SecureServingOptions[Svc, Store]) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.BindAddress, "secure.bind-address", s.BindAddress,
		"The IP address on which to listen for the --secure.bind-port port. The "+
			"associated interface(s) must be reachable by the rest of the engine, and by CLI/web "+
			"clients. If blank, all interfaces will be used (0.0.0.0).")

	fs.IntVar(&s.BindPort, "secure.bind-port", s.BindPort,
		"The port on which to serve HTTPS with authentication and authorization. If 0, "+
			"don't serve HTTPS at all.")

	fs.StringVar(&s.ServerCert.CertKey.CertFile, "secure.tls.cert-file", s.ServerCert.CertKey.CertFile,
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
			"after server cert). If HTTPS serving is enabled, and --secure.tls.cert-file and "+
			"--secure.tls.private-key-file are not provided, a certificate will be automatically generated.")

	fs.StringVar(&s.ServerCert.CertKey.KeyFile, "secure.tls.private-key-file", s.ServerCert.CertKey.KeyFile,
		"File containing the private key matching --secure.tls.cert-file.")

	fs.StringVar(&s.ServerCert.CertDirectory, "secure.tls.cert-dir", s.ServerCert.CertDirectory,
		"The directory where the TLS certs are located. "+
			"If --secure.tls.cert-file and --secure.tls.private-key-file are provided, "+
			"this flag will be ignored.")

	fs.StringVar(&s.ServerCert.PairName, "secure.tls.pair-name", s.ServerCert.PairName,
		"The name which will be used with --secure.tls.cert-dir to make a cert and key filenames. "+
			"It becomes <cert-dir>/<pair-name>.crt and <cert-dir>/<pair-name>.key")
}

func (s *SecureServingOptions[Svc, Store]) Complete() error {
	if s == nil || s.BindPort == 0 {
		return nil
	}

	// Complete BindAddress with default value if empty
	if s.BindAddress == "" {
		s.BindAddress = "0.0.0.0"
	}

	// If both cert and key files are not explicitly specified, generate them automatically
	if s.ServerCert.CertKey.CertFile == "" && s.ServerCert.CertKey.KeyFile == "" {
		// Use CertDirectory and PairName to generate certificate file paths
		if s.ServerCert.CertDirectory == "" {
			return errors.Errorf("--secure.tls.cert-dir: required flag is not set when cert-file and private-key-file are not provided")
		}
		if s.ServerCert.PairName == "" {
			return errors.Errorf("--secure.tls.pair-name: required flag is not set when cert-file and private-key-file are not provided")
		}

		s.ServerCert.CertKey.CertFile = filepath.Join(s.ServerCert.CertDirectory, s.ServerCert.PairName+".crt")
		s.ServerCert.CertKey.KeyFile = filepath.Join(s.ServerCert.CertDirectory, s.ServerCert.PairName+".key")
	}

	return nil
}
