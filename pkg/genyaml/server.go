package genyaml

import "github.com/ClessLi/component-base/pkg/genyaml/server"

func createServer(cfg *server.Config) (*server.GenerateYamlServer, error) {
	s, err := cfg.Complete().NewServer()
	if err != nil {
		return nil, err
	}
	return s, nil
}
