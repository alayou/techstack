package service

import (
	"fmt"

	cli "github.com/urfave/cli/v2"

	"github.com/alayou/techstack/daemon/service"

	"github.com/alayou/techstack/global"
)

type stopCommand struct {
	config service.Config
}

func (c *stopCommand) run(*cli.Context) error {
	fmt.Printf("stopping service %s\n", c.config.Name)
	s, err := service.New(c.config)
	if err != nil {
		return err
	}
	return s.Stop()
}

func registerStop() *cli.Command {
	c := new(stopCommand)
	c.config.ConfigFile = configPath()

	return &cli.Command{
		Name:        "stop",
		Description: "stop the service",
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
