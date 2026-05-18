package middleware

import (
	"context"
	"nft-auction-backend/internal/logger"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 前置处理
		start := time.Now()
		path := c.Request.URL.Path
		requestID := uuid.NewString()
		ctx := context.WithValue(c.Request.Context(), "requestID", requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Set("requestID", requestID)

		// 进入下一个处理函数
		c.Next()

		// 后置处理
		latency := time.Since(start)
		status := c.Writer.Status()
		logger.S().Infof("[%s %s] %s %d %v\n", requestID, c.Request.Method, path, status, latency)
	}
}
