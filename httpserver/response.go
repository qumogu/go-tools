package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Success 返回成功响应.
func Success(c *gin.Context, data interface{}) {
	content := gin.H{
		"code": 0,
		"desc": "success",
	}

	if data != nil {
		content["data"] = data
	}

	c.JSON(http.StatusOK, content)
}

// Failure 返回失败响应.
// trace 为底层详细错误.
func Failure(c *gin.Context, errCode int, errMsg string) {
	content := gin.H{
		"code": errCode,
		"desc": errMsg,
	}

	c.JSON(http.StatusOK, content)
}

// FailureWithhttpStatus 返回指定http状态码的失败响应.
func FailureWithHTTPStatus(c *gin.Context, httpStatus, errCode int, errMsg string) {
	content := gin.H{
		"code": errCode,
		"desc": errMsg,
	}
	c.JSON(httpStatus, content)
}
