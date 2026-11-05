package cli

import (
	"github.com/ejoja/shortcuts/internal/config"
	"github.com/ejoja/shortcuts/internal/shim"
	"github.com/ejoja/shortcuts/internal/xdg"
)

// service wires a config store and the resolved shim paths for one command.
type service struct {
	s       *Streams
	store   *config.Store
	cfgPath string
	shimDir string
}

func newService(s *Streams) (*service, error) {
	cfgPath, err := xdg.ConfigPath()
	if err != nil {
		return nil, err
	}
	shimDir, err := xdg.ShimDir()
	if err != nil {
		return nil, err
	}
	return &service{
		s:       s,
		store:   config.Open(cfgPath),
		cfgPath: cfgPath,
		shimDir: shimDir,
	}, nil
}

func (sv *service) load() (config.Config, error) {
	return sv.store.Load()
}

func (sv *service) manager(force bool) (*shim.Manager, error) {
	bin, err := shim.ResolveBinary()
	if err != nil {
		return nil, err
	}
	return &shim.Manager{Dir: sv.shimDir, Binary: bin, Force: force}, nil
}
