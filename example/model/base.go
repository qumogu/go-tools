package model

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CTX_USER_ID       = "user_id"
	CTX_USER_NAME     = "user_name"
	CTX_ORG_ID        = "org_id" // 等于 tenant_id
	CTX_IS_SUPERADMIN = "is_superadmin"
	CTX_GROUP_ID      = "group_id"
	CTX_CLIENT_IP     = "client_ip"
	CTX_USER_INFO     = "USERINFO"
)

type UserInfo struct {
	UserId       string `json:"id"`
	UserName     string `json:"name"`
	OrgId        string `json:"tenant_id"`
	GroupId      string `json:"group_id"`
	IsSuperAdmin int64  `json:"is_super_admin"`
}

type Base struct {
	// mongo主键id
	ID primitive.ObjectID `bson:"_id" json:"id"`

	OrgId string `json:"orgId" bson:"org_id"` // s 租户id

	// 创建
	CreatedUserId string `json:"createdUserId" bson:"created_user_id"` // 创建人ID
	CreatedUser   string `json:"createdUser" bson:"created_user"`      // 创建人姓名
	CreatedTime   int64  `json:"createdTime" bson:"created_time"`      // 创建时间

	// 修改
	UpdatedUserId string `json:"updatedUserId" bson:"updated_user_id"` // 修改人ID
	UpdatedUser   string `json:"updatedUser" bson:"updated_user"`      // 修改人姓名
	UpdatedTime   int64  `json:"updatedTime" bson:"updated_time"`      // 修改时间
}

func (b *Base) SetUpdate(userInfo UserInfo) {
	b.UpdatedTime = time.Now().Unix()
	b.UpdatedUser = userInfo.UserName
	b.UpdatedUserId = userInfo.UserId
}

func NewBaseWithUser(userInfo UserInfo) Base {
	return Base{
		ID:    primitive.NewObjectID(),
		OrgId: userInfo.OrgId,

		CreatedUserId: userInfo.UserId,
		CreatedUser:   userInfo.UserName,
		CreatedTime:   time.Now().Unix(),

		UpdatedUserId: userInfo.UserId,
		UpdatedUser:   userInfo.UserName,
		UpdatedTime:   time.Now().Unix(),
	}
}

func Gin2UserContext(c *gin.Context) (context.Context, UserInfo) {
	userId := c.GetString(CTX_USER_ID)
	userName := c.GetString(CTX_USER_NAME)
	orgId := c.GetString(CTX_ORG_ID)
	isSuperAdmin := c.GetInt64(CTX_IS_SUPERADMIN)
	groupId := c.GetString(CTX_GROUP_ID)
	clientIP := c.GetString(CTX_CLIENT_IP)

	ctx := context.WithValue(context.Background(), CTX_USER_ID, userId)
	ctx = context.WithValue(ctx, CTX_USER_NAME, userName)
	ctx = context.WithValue(ctx, CTX_ORG_ID, orgId)
	ctx = context.WithValue(ctx, CTX_IS_SUPERADMIN, isSuperAdmin)
	ctx = context.WithValue(ctx, CTX_GROUP_ID, groupId)
	ctx = context.WithValue(ctx, CTX_CLIENT_IP, clientIP)

	userInfo := UserInfo{
		UserId:   userId,
		UserName: userName,
		OrgId:    orgId,
		GroupId:  groupId,
		// IsSuperAdmin: isSuperAdmin,
	}

	ctx = context.WithValue(ctx, CTX_USER_INFO, userInfo)

	return ctx, userInfo
}
