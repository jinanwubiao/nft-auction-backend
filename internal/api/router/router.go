package router

import (
	"nft-auction-backend/internal/api/middleware"
	v1 "nft-auction-backend/internal/api/v1"
	"nft-auction-backend/internal/service/svc"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(svcCtx *svc.ServerCtx) *gin.Engine {
	gin.ForceConsoleColor()
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	//恢复
	r.Use(gin.Recovery())
	//日志记录
	r.Use(middleware.LoggerMiddleware())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "X-CSRF-Token", "Authorization", "AccessToken", "Token"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "X-GW-Error-Code", "X-GW-Error-Message"},
		AllowCredentials: true,
		MaxAge:           1 * time.Hour,
	}))
	
	loadV1(r, svcCtx)

	// setup swagger ui server side
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	return r
}

func loadV1(r *gin.Engine, svcCtx *svc.ServerCtx) {
	apiV1 := r.Group("/api/v1")
	user := apiV1.Group("/user")
	{
		user.GET("/login", v1.UserLoginHandler(svcCtx))
	}
}