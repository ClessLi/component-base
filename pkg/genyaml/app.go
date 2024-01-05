package genyaml

import (
	"github.com/ClessLi/component-base/pkg/app"
	"github.com/ClessLi/component-base/pkg/genyaml/options"
	"github.com/ClessLi/component-base/pkg/genyaml/server"
	"github.com/spf13/cobra"
)

const commandDesc = `An application used to generate yaml configurations for applications.`

func NewGenYamlApp(yamlname string, command *cobra.Command) *app.App {
	opts := options.NewGenYamlOptions()
	application := app.NewApp("generate-yaml",
		yamlname,
		app.WithOptions(opts),
		app.WithDescription(commandDesc),
		app.WithDefaultValidArgs(),
		app.WithNoConfig(),
		app.WithRunFunc(run(opts, command)),
	)

	return application
}

func run(opts *options.GenYamlOptions, command *cobra.Command) app.RunFunc {
	return func(yamlname string) error {
		cfg, err := CreateConfigFromOptions(opts)
		if err != nil {
			return err
		}

		return Run(cfg, command)
	}
}

func CreateConfigFromOptions(opts *options.GenYamlOptions) (c *server.Config, err error) {
	c = server.NewConfig()
	err = opts.ApplyTo(c)
	return
}
