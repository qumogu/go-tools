package model

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qumogu/go-tools/mongodb"
)

type Message struct {
	Base    `bson:",inline"`
	Title   string `json:"title" bson:"title"`
	Content string `json:"content" bson:"content"`
	Url     string `json:"url" bson:"url"`
}

// 实现crud.param接口
func (m Message) NewModel(ctx *gin.Context) interface{} {
	return Message{}
}

// 实现crud.param接口
func (m Message) NewCreate(ctx *gin.Context) interface{} {
	_, userInfo := Gin2UserContext(ctx)
	config := Message{
		Base: NewBaseWithUser(userInfo),
	}

	// fmt.Printf("%+v\n", config)
	return &config
}

// 实现crud.param接口
func (m Message) NewSearch(ctx *gin.Context) interface{} {
	return &ConfigurationSearch{}
}

// 实现crud.param接口
func (m Message) NewUpdate(ctx *gin.Context) interface{} {
	return MessageUpdate{}
}

// 实现crud.param接口
func (m Message) NewListResult(ctx *gin.Context) interface{} {
	return &[]Message{}
}

// 实现crud.param接口
// columns := []ExcelColumn{{Key: "name", Name: "姓名"}, {Key: "age", Name: "年龄"}, {Key: "location", Name: "住址"}, {Key: "update_time", Name: "更新时间", ExportFormat: timeStampExportFormat}}
func (m Message) ExcelColumns() []mongodb.ExcelColumn {
	return []mongodb.ExcelColumn{
		{Key: "seq", Name: "序号"},
		{Key: "station_seq", Name: "站序号"},
		{Key: "sc_seq", Name: "监控索引号"},
		{Key: "device_name", Name: "设备名称"},
		{Key: "signal_type", Name: "信号类型"},
		{Key: "point_position", Name: "点位"},
	}
}

// 实现crud.param接口
func (m Message) ExcelName() string {
	return "配置明细表" + time.Now().Format("20060102150405")
}
