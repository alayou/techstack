// Package daemon implements the daemon runner.

package daemon

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alayou/techstack/httpd/taskmgr"
	"github.com/alayou/techstack/utils"
	"github.com/gorpher/gone/cache"
	"github.com/gorpher/gone/logger"
	"github.com/rs/zerolog/log"

	"github.com/alayou/techstack/httpd"

	"github.com/alayou/techstack/global"
	"golang.org/x/sync/errgroup"
)

const sender = "daemon"

type Daemon struct {
	exitFunc   context.CancelFunc
	checksum   string
	httpServer *http.Server // http服务
	tlsConfig  *tls.Config  // 证书配置
}

func NewDaemon() *Daemon {
	return &Daemon{}
}

func (s *Daemon) Prepare() error {
	var (
		err error
	)
	s.checksum, err = utils.LoadYAMLConfig(global.Config.ConfigFile, &global.Config)
	if err != nil {
		log.Warn().Str("sender", sender).Err(err).
			Str("ConfigFile", global.Config.ConfigFile).Msg("cannot load config file.")
	}
	logger.SetZerolog(
		logger.WithFileName(global.Config.LogFile),
		logger.WithLogConfig(&logger.LogConfig{
			LogLevel:       global.Config.LogLevel,
			LogWithConsole: global.Config.Debug,
		}))
	return err
}

// Run runs the service and blocks until complete.
func (s *Daemon) Run(parentCtx context.Context) {
	var (
		g   errgroup.Group
		err error
	)
	ctx, cancelFunc := context.WithCancel(parentCtx)
	s.exitFunc = cancelFunc
	// 初始化证书
	if len(global.Config.Ca) > 0 {
		s.tlsConfig = setupMutualTLS(global.Config.Ca)
	}
	if len(global.Config.Cert) > 0 && len(global.Config.Key) > 0 {
		s.tlsConfig = setupMutualTLSFromFile(global.Config.Cert, global.Config.Key)
	}
	serv := httpd.NewServer(ctx, cache.NewMemoryCache(), global.WebFS)
	s.httpServer = &http.Server{
		Addr:        global.Config.Addr,
		Handler:     serv,
		ReadTimeout: 5 * time.Second,
	}
	if s.tlsConfig != nil {
		s.httpServer.TLSConfig = s.tlsConfig
	}

	// 启动http服务
	g.Go(func() error {
		logger.Info(sender, "Listening on Addr %s", global.Config.Addr)
		if s.tlsConfig != nil {
			addr := global.Config.Addr
			if addr == "" {
				addr = ":https"
			}
			var l net.Listener
			l, err = net.Listen("tcp", addr)
			if err != nil {
				cancelFunc()
				return err
			}
			ln := tls.NewListener(l, s.tlsConfig)
			err = s.httpServer.Serve(ln)
			if err != nil {
				cancelFunc()
			}
			return err
		}
		err = s.httpServer.ListenAndServe()
		if err != nil {
			cancelFunc()
		}
		return err
	})

	// 启动后台任务 Worker Pool
	taskmgr.StartWorkerPool(ctx)

	// 优雅的关闭http和grpc服务
	g.Go(func() error {
		defer func() {
			cancelFunc()
			logger.Debug(sender, "优雅关闭http和grpc服务协程-已退出")
		}()
		<-ctx.Done()
		return s.httpServer.Shutdown(context.Background()) // nolint
	})

	err = g.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("shutting down the server")
	}

}

func setupMutualTLS(ca string) *tls.Config {
	clientCACert, err := os.ReadFile(filepath.Clean(ca))
	if err != nil {
		log.Fatal().Err(err).Msg("读取证书文件失败")
	}

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCACert)

	tlsConfig := &tls.Config{
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                clientCertPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}

	return tlsConfig
}

func setupMutualTLSFromFile(certFile, keyFile string) *tls.Config {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal().Err(err).Msg("读取证书文件失败")
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{certificate},
		Rand:         rand.Reader,
	}
}
