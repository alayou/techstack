package command

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/alayou/techstack/global"
)

func registerVersion(app *cli.App) {
	app.Commands = append(app.Commands,
		&cli.Command{
			Name:  "version",
			Usage: "show app version",
			Action: func(c *cli.Context) error {
				fmt.Print(version())
				return nil
			},
		})
}

func version() string {
	return fmt.Sprintf("%s has version %s built from %s on %s\n", global.AppName, global.Version, global.Revision, global.RBuiltAt)
}
