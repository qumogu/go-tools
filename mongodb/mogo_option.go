package mongodb

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ PageOrderIntfc = (*PageOrder)(nil)

type PageOrderIntfc interface {
	GetMongoFindOptions() *options.FindOptions
}

// 分页参数.
type Page struct {
	Limit  *int64 `form:"limit,default=10" json:"limit" bson:"limit"`   // 每页大小, 默认10
	Offset *int64 `form:"offset,default=0" json:"offset" bson:"offset"` // 偏移量，从0开始， 默认0
}

func (p Page) GetMongoFindOptions() *options.FindOptions {
	opt := &options.FindOptions{}

	if p.Offset != nil {
		opt.SetSkip(*p.Offset)
	}

	if p.Limit != nil {
		opt.SetLimit(*p.Limit)
	}

	return opt
}

// 排序.
type Order struct {
	OrderBy string `form:"order_by" json:"order_by" bson:"order_by"` // 排序字段，默认 _id
	Asc     bool   `form:"asc" json:"asc" bson:"asc"`                // true： 升序， false：降序, 默认降序
}

// 分页与排序.
// 分页传0是查询所有.
type PageOrder struct {
	Page
	Order // TODO: 支持切片
}

// 分页函数返回 mongoDB 分页option, order排序.
func (p PageOrder) GetMongoFindOptions() *options.FindOptions {
	opt := &options.FindOptions{}

	if p.Offset != nil {
		opt.SetSkip(*p.Offset)
	}

	if p.Limit != nil {
		opt.SetLimit(*p.Limit)
	}

	if p.OrderBy == "" {
		defaultField := "_id"
		p.OrderBy = defaultField
	}

	if p.Asc {
		opt.Sort = bson.D{primitive.E{Key: p.OrderBy, Value: 1}}
		opt.SetCollation(&options.Collation{
			Locale: "zh",
		})
	} else {
		opt.Sort = bson.D{primitive.E{Key: p.OrderBy, Value: -1}}
		opt.SetCollation(&options.Collation{
			Locale: "zh",
		})
	}

	return opt
}

func NewPageOrder(limit, offset int64) PageOrder {
	return PageOrder{
		Page: Page{
			Limit:  &limit,
			Offset: &offset,
		},
	}
}
