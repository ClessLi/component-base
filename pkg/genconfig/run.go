package genconfig

import (
	"github.com/ClessLi/component-base/pkg/genconfig/server"
	"github.com/spf13/cobra"
)

func Run(cfg *server.Config, command *cobra.Command) error {
	s, err := createServer(cfg)
	if err != nil {
		return err
	}
	return s.GenYaml(command)
}
