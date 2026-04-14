package service

import cli "github.com/urfave/cli/v2"

// Register registers the command.
func Register(app *cli.App) {
	var commands = []*cli.Command{
		registerInstall(),
		registerStart(),
		registerStop(),
		registerUninstall(),
		registerRun(),
	}
	app.Commands = append(app.Commands,
		&cli.Command{
			Name:        "service",
			Usage:       "manages the runner service",
			Subcommands: commands,
		},
	)

}
