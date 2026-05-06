package app

import (
	"nft-auction-backend/internal/config"
	"nft-auction-backend/internal/service/svc"

	"github.com/gin-gonic/gin"
)

type App struct {
	config    *config.Config
	router    *gin.Engine
	serverCtx *svc.ServerCtx
}

func NewApp(config *config.Config, router *gin.Engine, serverCtx *svc.ServerCtx) *App {
	return &App{
		config:    config,
		router:    router,
		serverCtx: serverCtx,
	}
}

func (a *App) run() {

}
