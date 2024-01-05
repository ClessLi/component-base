//go:build !viper_yaml2
// +build !viper_yaml2

package server

type cmdOption struct {
	Name         string
	Shorthand    string `yaml:",omitempty"`
	DefaultValue string `yaml:"default_value,omitempty"`
	Usage        string `yaml:",omitempty"`
}

type cmdDoc struct {
	Name             string
	Synopsis         string      `yaml:",omitempty"`
	Description      string      `yaml:",omitempty"`
	Options          []cmdOption `yaml:",omitempty"`
	InheritedOptions []cmdOption `yaml:"inherited_options,omitempty"`
	Example          string      `yaml:",omitempty"`
	SeeAlso          []string    `yaml:"see_also,omitempty"`
}

type YamlInfo struct {
	ParentName string
	DocsDir    string
}

type Config struct {
	YamlInfo *YamlInfo
}

func NewConfig() *Config {
	return &Config{new(YamlInfo)}
}

type CompletedConfig struct {
	*Config
}

func (c *Config) Complete() CompletedConfig {
	return CompletedConfig{c}
}

func (c CompletedConfig) NewServer() (*GenConfigServer, error) {
	return &GenConfigServer{YamlInfo: c.YamlInfo}, nil
}
