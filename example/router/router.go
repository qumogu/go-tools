package router

import (
	"github.com/qumogu/go-tools/example/config"
	"github.com/qumogu/go-tools/example/model"
	"github.com/qumogu/go-tools/mongodb"

	"github.com/gin-gonic/gin"
)

func InitRouter(r *gin.Engine) {
	msgGrp := r.Group("/api/v1/message")
	msgGrp.Use(UserInfo)
	msg := mongodb.NewCrud(config.Conf.Mongo.Database, "message", model.Message{})
	mongodb.CRUD(msgGrp, "", msg)
}

func UserInfo(c *gin.Context) {
	c.Set(model.CTX_USER_ID, "1003")
	c.Set(model.CTX_USER_NAME, "jack")
	c.Set(model.CTX_ORG_ID, "test-org-101") // 租户id, 用于动态连接的数据库
	c.Next()
}
