package options

import (
	"github.com/ClessLi/component-base/pkg/genyaml/server"
	cliflag "github.com/marmotedu/component-base/pkg/cli/flag"
	"github.com/marmotedu/errors"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
	"strings"
)

const (
	flagParentName    = "parent-name"
	flagOutputDirPath = "output-dir-path"
)

type GenYamlOptions struct {
	ParentName    string `json:"parent-name" mapstructure:"parent-name"`
	OutputDirPath string `json:"output-dir-path" mapstructure:"output-dir-path"`
}

func (g *GenYamlOptions) Flags() (fss cliflag.NamedFlagSets) {
	g.AddFlags(fss.FlagSet("generate yaml"))
	return fss
}

func NewGenYamlOptions() *GenYamlOptions {
	return &GenYamlOptions{}
}

func (g *GenYamlOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&g.ParentName, flagParentName, g.ParentName, "Set parent field for generated yaml.")

	fs.StringVar(&g.OutputDirPath, flagOutputDirPath, g.OutputDirPath, "Set the output file path for generated yaml.")

}

func (g *GenYamlOptions) Validate() []error {
	var errs []error
	if len(strings.TrimSpace(g.OutputDirPath)) == 0 {
		errs = append(errs, errors.Errorf("--%v is null.", flagOutputDirPath))
	} else {
		s, err := os.Stat(g.OutputDirPath)
		if err != nil && !os.IsExist(err) {
			errs = append(errs, errors.Errorf("--%v validate failed, cased by: %v", flagOutputDirPath, err))
		} else if !s.IsDir() {
			errs = append(errs, errors.Errorf("--%v validate failed, cased by: the output dir `%v` is a exist file, not a directory", flagOutputDirPath, g.OutputDirPath))
		}
	}

	return errs
}

func (g *GenYamlOptions) ApplyTo(c *server.Config) error {
	outputDirPath, err := filepath.Abs(g.OutputDirPath)
	if err != nil {
		return err
	}

	c.YamlInfo.DocsDir = filepath.Clean(outputDirPath)
	c.YamlInfo.ParentName = g.ParentName

	return nil
}
