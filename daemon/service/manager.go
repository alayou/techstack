package service

import (
	"context"

	"github.com/gorpher/gone/osutil"

	"github.com/alayou/techstack/global"

	"github.com/rs/zerolog/log"

	"github.com/alayou/techstack/daemon"

	"github.com/kardianos/service"
)

var nocontext = context.Background()

// a manager manages the service lifecycle.
type manager struct {
	cancel     context.CancelFunc
	configFile *string
}

// Start starts the service in a separate go routine.
func (m *manager) Start(service.Service) error {
	ctx, cancel := context.WithCancel(nocontext)
	m.cancel = cancel
	global.Config.ConfigFile = *m.configFile

	// listen for termination signals to gracefully shutdown
	// the runner daemon.
	ctx = osutil.WithContextFunc(ctx, func() {
		log.Info().Msg("received signal, terminating process")
		cancel()
	})

	daem := daemon.NewDaemon()
	err := daem.Prepare()
	if err != nil {
		return err
	}
	go daem.Run(ctx)
	return nil
}

// Stop stops the service.
func (m *manager) Stop(service.Service) error {
	m.cancel()
	return nil
}
