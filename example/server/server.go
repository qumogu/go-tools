package server

import (
	"context"
	"time"

	"github.com/qumogu/go-tools/example/config"
	"github.com/qumogu/go-tools/example/router"
	"github.com/qumogu/go-tools/httpserver"
	"github.com/qumogu/go-tools/logger"
	"github.com/qumogu/go-tools/svrctrl"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Init() {
	config.Parse()
	s.ctx, s.cancel = context.WithCancel(context.Background())

	logger.Info("server init")

}

func (s *Server) Run() {
	// 启动http服务
	s.httpServer()

	// 处理 SIGTERM 和 SIGINT
	svrctrl.Trap(s.cancel)

	<-s.ctx.Done()
	logger.Info("server exiting")
}

func (s *Server) httpServer() {
	logger.Info("start http server ...")

	errGroup, ctx := errgroup.WithContext(s.ctx)
	r := httpserver.NewRouter(config.Conf.ServerRunMode, config.Conf.Profile)
	router.InitRouter(r)
	svrctrl.GoAndRestartOnError(ctx, errGroup, "linkage-http", func() error {
		return httpserver.Run(ctx, r, config.Conf.HttpPort, time.Second*time.Duration(config.Conf.GracefulTimeout))
	})
}
