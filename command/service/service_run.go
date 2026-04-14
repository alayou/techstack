package service

import (
	"github.com/alayou/techstack/daemon/service"
	"github.com/alayou/techstack/global"
	cli "github.com/urfave/cli/v2"
)

type runCommand struct {
	config service.Config
}

func (c *runCommand) run(*cli.Context) error {
	s, err := service.New(c.config)
	if err != nil {
		return err
	}
	return s.Run()
}

func registerRun() *cli.Command {
	c := new(runCommand)
	c.config.ConfigFile = configPath()

	return &cli.Command{
		Name:        "run",
		Description: "run the service",
		Action:      c.run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "service name",
				Value:       global.AppName,
				Destination: &c.config.Name,
			},
			&cli.StringFlag{
				Name:        "desc",
				Usage:       "service description",
				Value:       global.AppDesc,
				Destination: &c.config.Desc,
			},
			&cli.StringFlag{
				Name:        "username",
				Usage:       "windows account username",
				Value:       "",
				Destination: &c.config.Username,
			},
			&cli.StringFlag{
				Name:        "password",
				Usage:       "windows account password",
				Value:       "",
				Destination: &c.config.Password,
			},
			&cli.StringFlag{
				Name:        "config",
				Usage:       "service configuration file",
				Value:       c.config.ConfigFile,
				Destination: &c.config.ConfigFile,
			},
		},
	}
}
