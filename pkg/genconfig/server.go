package genconfig

import "github.com/ClessLi/component-base/pkg/genconfig/server"

func createServer(cfg *server.Config) (*server.GenConfigServer, error) {
	s, err := cfg.Complete().NewServer()
	if err != nil {
		return nil, err
	}
	return s, nil
}
