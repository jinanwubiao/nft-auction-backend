package app

import (
	"context"
	"flag"
	"net/http"
	"nft-auction-backend/internal/api/router"
	"nft-auction-backend/internal/config"
	"nft-auction-backend/internal/indexer"
	"nft-auction-backend/internal/logger"
	"nft-auction-backend/internal/service/svc"
	"time"

	"github.com/gin-gonic/gin"
)

type App struct {
	cfg     *config.Config
	httpSrv *http.Server
	router  *gin.Engine
	svcCtx  *svc.ServerCtx
	idx     *indexer.Indexer
}

const (
	defaultConfigPath = "./config/config.toml"
)

func NewApp() (*App, error) {
	//加载配置
	conf := flag.String("conf", defaultConfigPath, "conf file path")
	flag.Parse()
	c, err := config.Load(*conf)
	if err != nil {
		return nil, err
	}
	//初始化服务
	serverCtx, err := svc.NewServiceContext(c)
	if err != nil {
		return nil, err
	}
	//初始化索引器
	idx, err := indexer.NewIndexer(c.Eth, serverCtx)
	if err != nil {
		return nil, err
	}
	//初始化路由
	r := router.NewRouter(serverCtx)
	//自定义httpServer
	httpServer := &http.Server{
		Addr:         c.Server.Port,
		Handler:      r,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return &App{
		cfg:     c,
		httpSrv: httpServer,
		router:  r,
		svcCtx:  serverCtx,
		idx:     idx,
	}, nil
}

func (app *App) Run() error {
	//启动索引器
	go app.idx.Start()
	//启动服务
	if err := app.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (app *App) Shutdown(ctx context.Context) error {
	//关停服务
	err := app.httpSrv.Shutdown(ctx)
	//关闭索引器
	app.idx.Stop()
	//同步日志缓冲区
	logger.Sync()
	return err
}
