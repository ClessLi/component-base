// Package server provides generic HTTP and gRPC server implementations with
// configuration-driven setup, lifecycle management, and graceful shutdown support.
package server

import (
	"net"
	"path/filepath"
	"strconv"
	"strings"

	logV1 "github.com/ClessLi/component-base/pkg/log/v1"
	"github.com/marmotedu/component-base/pkg/util/homedir"
	"github.com/spf13/viper"
)

const (
	// RecommendedHomeDir defines the default directory used to place all service configurations.
	RecommendedHomeDir = ".component"

	// RecommendedEnvPrefix defines the ENV prefix used by all services.
	RecommendedEnvPrefix = "COMPONENT"
)

// CertKey contains configuration items related to certificate.
type CertKey struct {
	// CertFile is a file containing a PEM-encoded certificate, and possibly the complete certificate chain
	CertFile string
	// KeyFile is a file containing a PEM-encoded private key for the certificate specified by CertFile
	KeyFile string
}

// SecureServingInfo holds configuration of the TLS server.
type SecureServingInfo struct {
	BindAddress string
	BindPort    int
	CertKey     CertKey
}

// Address join host IP address and host port number into an address string, like: 0.0.0.0:8443.
func (s *SecureServingInfo) Address() string {
	return net.JoinHostPort(s.BindAddress, strconv.Itoa(s.BindPort))
}

// InsecureServingInfo holds configuration of the insecure http server.
type InsecureServingInfo struct {
	BindAddress string
	BindPort    int
}

// Address join host IP address and host port number into an address string.
func (i *InsecureServingInfo) Address() string {
	return net.JoinHostPort(i.BindAddress, strconv.Itoa(i.BindPort))
}

// LoadConfig reads in config file and ENV variables if set.
func LoadConfig(cfg string, defaultName string, homeDir string, envPrefix string) {
	if cfg != "" {
		viper.SetConfigFile(cfg)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath(filepath.Join(homedir.HomeDir(), homeDir))
		viper.AddConfigPath("/etc/component")
		viper.SetConfigName(defaultName)
	}

	// Use config file from the flag.
	viper.SetConfigType("yaml")   // set the type of the configuration to yaml.
	viper.AutomaticEnv()          // read in environment variables that match.
	viper.SetEnvPrefix(envPrefix) // set ENVIRONMENT variables prefix.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		logV1.Warnf("WARNING: viper failed to discover and load the configuration file: %s", err.Error())
	}
}
