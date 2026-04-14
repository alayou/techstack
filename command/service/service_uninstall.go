package service

import (
	"fmt"

	cli "github.com/urfave/cli/v2"

	"github.com/alayou/techstack/daemon/service"
	"github.com/alayou/techstack/global"
)

type uninstallCommand struct {
	config service.Config
}

func (c *uninstallCommand) run(*cli.Context) error {
	fmt.Printf("uninstalling service %s\n", c.config.Name)
	s, err := service.New(c.config)
	if err != nil {
		return err
	}
	return s.Uninstall()
}

func registerUninstall() *cli.Command {
	c := new(uninstallCommand)
	c.config.ConfigFile = configPath()

	return &cli.Command{
		Name:        "uninstall",
		Description: "uninstall the service",
		Action:      c.run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "service name",
				Value:       global.AppName,
				Destination: &c.config.Name,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "desc",
				Usage:       "service description",
				Value:       global.AppDesc,
				Destination: &c.config.Desc,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "username",
				Usage:       "windows account username",
				Value:       "",
				Destination: &c.config.Username,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "password",
				Usage:       "windows account password",
				Value:       "",
				Destination: &c.config.Password,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "config",
				Usage:       "service configuration file",
				Value:       c.config.ConfigFile,
				Destination: &c.config.ConfigFile,
				Required:    false,
			},
		},
	}
}
