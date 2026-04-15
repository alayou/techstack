package command

import (
	"context"

	"github.com/gorpher/gone/osutil"
	"github.com/urfave/cli/v2"

	"github.com/rs/zerolog/log"

	"github.com/alayou/techstack/daemon"
	"github.com/alayou/techstack/global"
)

type daemonCommand struct {
}

func (c *daemonCommand) run(pCtx *cli.Context) error {
	ctx, cancel := context.WithCancel(pCtx.Context)
	defer cancel()

	// listen for termination signals to gracefully shutdown
	// the runner daemon.
	ctx = osutil.WithContextFunc(ctx, func() {
		log.Info().Msg("received signal, terminating process")
		cancel()
	})

	daem := daemon.NewDaemon()
	if err := daem.Prepare(); err != nil {
		return err
	}
	daem.Run(ctx)
	return nil
}

func registerDaemon(app *cli.App) {
	c := new(daemonCommand)
	app.Action = c.run
	app.Commands = append(app.Commands, &cli.Command{
		Name:   "daemon",
		Usage:  "starts the runner daemon",
		Action: c.run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"conf"},
				Value:       global.Config.ConfigFile,
				Usage:       "service configuration file",
				Destination: &global.Config.ConfigFile,
				EnvVars:     []string{"TECHSTACK_CONFIG"},
			},
			&cli.StringFlag{
				Name:        "addr",
				Value:       global.Config.Addr,
				Usage:       "http listen address",
				Destination: &global.Config.Addr,
				EnvVars:     []string{"TECHSTACK_ADDR"},
			},
			&cli.StringFlag{
				Name:        "cert",
				Value:       global.Config.Cert,
				Usage:       "cert",
				Destination: &global.Config.Cert,
				EnvVars:     []string{"TECHSTACK_CERT"},
			},
			&cli.StringFlag{
				Name:        "key",
				Value:       global.Config.Key,
				Usage:       "key",
				Destination: &global.Config.Key,
				EnvVars:     []string{"TECHSTACK_KEY"},
			},
			&cli.StringFlag{
				Name:        "ca",
				Value:       global.Config.Ca,
				Usage:       "ca",
				Destination: &global.Config.Ca,
				EnvVars:     []string{"TECHSTACK_CA"},
			},
			&cli.StringFlag{
				Name:        "logFile",
				Value:       global.Config.LogFile,
				Usage:       "log file",
				Destination: &global.Config.LogFile,
				EnvVars:     []string{"TECHSTACK_LOG_FILE"},
			},
			&cli.StringFlag{
				Name:        "loglevel",
				Value:       global.Config.LogLevel,
				Usage:       "log level",
				Destination: &global.Config.LogLevel,
				EnvVars:     []string{"TECHSTACK_LOG_LEVEL"},
			},
		},
	})
}
