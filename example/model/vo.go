package model

type ConfigurationSearch struct {
	Seq              *string  `json:"seq" form:"seq" bson:"seq"`                                              // 序号
	StationSeq       *int     `json:"station_seq" form:" station_seq" bson:"station_seq"`                     // 站序号
	SCIndex          *int     `json:"sc_index" form:"sc_index" bson:"sc_index"`                               // 监控索引号, 用于数据的唯一标识
	DeviceName       *string  `json:"device_name" form:"device_name" bson:"device_name"`                      // 设备名称
	SignalType       *string  `json:"signal_type" form:"signal_type" bson:"signal_type"`                      // 信号类型, 遥控 1遥信 2遥测
	PointPositionIDs []string `json:"point_position_ids" form:"point_position_ids" bson:"point_position_ids"` // 点位ID 数组
	Priority         *int     `json:"priority" form:"priority" bson:"priority"`                               // 优先级
	Limit            *int     `json:"limit" form:"limit" bson:"limit"`
	Offset           *int     `json:"offset" form:"offset" bson:"offset"`
}

type MessageUpdate struct {
	Title   *string `json:"title" form:"title" bson:"title"`       // 序号
	Content *string `json:"content" form:"content" bson:"content"` // 设备名称
}
