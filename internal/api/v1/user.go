package v1

import (
	"nft-auction-backend/internal/service/svc"
	resp "nft-auction-backend/internal/util"

	"github.com/gin-gonic/gin"
)

func UserLoginHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp.Ok(c, "login test")
	}
}
