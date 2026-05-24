package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeSuccess     int = 0
	CodeFail        int = 1
	CodeSystemError int = -1

	// User module 10xxx
	CodeUserNotFound int = 10001
	CodePasswordErr  int = 10002
	CodeAuthExpired  int = 10004

	// Business module 20xxx
	CodeResourceGone int = 20001
	CodeLimitReached int = 20002
)

type Response struct {
	Code    int    `json:"code"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

type PageResult struct {
	Start      uint  `json:"start"`
	Limit      uint  `json:"limit"`
	Data       any   `json:"data"`
	TotalCount int64 `json:"totalCount"`
}

type UserResp struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

func Msg(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{Code: CodeSuccess, Message: message})
}

func Ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: CodeSuccess, Data: data, Message: "ok"})
}

func Fail(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{Code: CodeFail, Message: message})
}

func Result(c *gin.Context, httpCode int, businessCode int, message string) {
	c.JSON(httpCode, Response{Code: businessCode, Message: message})
}

func Err(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, Response{Code: http.StatusInternalServerError, Message: msg})
}

func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{Code: CodeFail, Message: message})
}

func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{Code: CodeFail, Message: message})
}

func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{Code: CodeFail, Message: message})
}
