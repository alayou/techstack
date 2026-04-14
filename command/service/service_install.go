package service

import (
	"github.com/gorpher/gone/osutil"
	cli "github.com/urfave/cli/v2"

	"os"
	"path/filepath"

	"github.com/alayou/techstack/daemon/service"
	"github.com/alayou/techstack/global"
	"github.com/rs/zerolog/log"
)

type installCommand struct {
	config service.Config
}

func (c *installCommand) run(ctx *cli.Context) error {
	log.Debug().Msgf("read configuration %s\n", c.config.ConfigFile)
	log.Debug().Msgf("installing service %s\n", c.config.Name)
	if !osutil.FileExist(c.config.ConfigFile) {
		if err := os.MkdirAll(filepath.Dir(c.config.ConfigFile), os.ModeDir); err != nil {
			return err
		}
		err := os.WriteFile(c.config.ConfigFile, []byte(""), os.FileMode(0666))
		if err != nil {
			return err
		}
	}
	s, err := service.New(c.config)
	if err != nil {
		return err
	}
	return s.Install()
}

func registerInstall() *cli.Command {
	c := new(installCommand)
	c.config.ConfigFile = configPath()
	return &cli.Command{
		Name:        "install",
		Description: "install the service",
		Action:      c.run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "name",
				Usage:       "install the service",
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
