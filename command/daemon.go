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
				Name:  "conf",
				Value: global.Config.ConfigFile,
				Usage: "load the environment variable file",
			},
			&cli.StringFlag{
				Name:  "addr",
				Value: global.Config.Addr,
				Usage: "http listen address",
			},
			&cli.StringFlag{
				Name:  "cert",
				Value: global.Config.Cert,
				Usage: "cert",
			},
			&cli.StringFlag{
				Name:  "key",
				Value: global.Config.Key,
				Usage: "key",
			},
			&cli.StringFlag{
				Name:  "ca",
				Value: global.Config.Ca,
				Usage: "ca",
			},
			&cli.StringFlag{
				Name:  "logFile",
				Value: global.Config.LogFile,
				Usage: "log file",
			},
			&cli.StringFlag{
				Name:  "loglevel",
				Value: global.Config.LogLevel,
				Usage: "log level",
			},
		},
	})
}
