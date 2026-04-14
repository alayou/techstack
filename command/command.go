package command

import (
	"log"

	"github.com/alayou/techstack/command/service"

	"os"

	"github.com/alayou/techstack/global"
	cli "github.com/urfave/cli/v2"
)

func Command() {
	app := &cli.App{
		Name:        global.AppName,
		Description: global.AppDesc,
		Version:     version(),
		Commands:    []*cli.Command{},
	}
	registerDaemon(app)
	registerVersion(app)
	service.Register(app)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
