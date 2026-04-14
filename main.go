package main

import (
	"embed"

	"github.com/alayou/techstack/command"
	"github.com/alayou/techstack/global"
	env "github.com/caarlos0/env/v6"
	"github.com/rs/zerolog/log"
)

var version = "unstacked"
var revision = "unstacked"
var builtAt = "unstacked"

//go:embed web
var WebFS embed.FS

func main() {
	global.Version = version
	global.Revision = revision
	global.RBuiltAt = builtAt
	global.WebFS = WebFS
	if err := env.Parse(&global.Config); err != nil {
		log.Err(err).Msg("解析环境变量失败")
	}
	command.Command()
}
