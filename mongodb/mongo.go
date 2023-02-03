package mongodb

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	DefaultMongoMgr  *MongoManager
	mongoMgrInitOnce sync.Once
)

func InitMongoManager(conf MongoConf) {
	mongoMgrInitOnce.Do(func() {
		DefaultMongoMgr = NewMongoManager(conf)
	})
}

type MongoConf struct {
	URI      string `json:"-"`                          // mongo server uri
	Addr     string `json:"addr" yaml:"ADDR"`           // ip:port
	Database string `json:"-"`                          // 数据库名称
	UserName string `json:"user_name" yaml:"USER_NAME"` // 用户名
	PWD      string `json:"pwd" yaml:"PWD"`             // 密码
}

// MongoConnMgr mongodb连接管理.
// 多租户下(目前是数据隔离), 不同租户连接的数据库不同.
type MongoManager struct {
	mongoClients sync.Map // FIXME: 租户数量不断增加，Map的数量也不断增加
	lock         sync.RWMutex
	conf         MongoConf
}

func NewMongoManager(conf MongoConf) *MongoManager {
	return &MongoManager{
		conf: conf,
	}
}

// 所有租户同一把锁.
// 数据库名称一般为租户ID
func (m *MongoManager) GetDB(dbName string) (*mongo.Database, error) {
	client, ok := m.mongoClients.Load(dbName)
	if !ok {
		m.lock.Lock()
		defer m.lock.Unlock()

		client, ok = m.mongoClients.Load(dbName)
		if ok {
			c, ok := client.(*mongo.Database)
			if !ok {
				return nil, fmt.Errorf("client convert to mongo.Database failed")
			}

			return c, nil
		}

		newClient, err := m.newDB(dbName)
		if err != nil {
			return nil, err
		}

		m.mongoClients.Store(dbName, newClient)

		return newClient, nil
	}

	// 每次获取客户端时ping一次连接是否正常, 如果不正常则删除当前k,v, 然后重新设置并返回新的连接
	if err := client.(*mongo.Database).Client().Ping(context.TODO(), readpref.Primary()); err != nil {
		m.mongoClients.Delete(dbName)

		newClient, err := m.newDB(dbName)
		if err != nil {
			return nil, err
		}

		m.mongoClients.Store(dbName, newClient)

		return newClient, nil
	}

	return client.(*mongo.Database), nil
}

func (m *MongoManager) newDB(tenantID string) (*mongo.Database, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(m.conf.URI))
	if err != nil {
		fmt.Println("构建数据库client异常: ", err)

		return nil, err
	}

	if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
		fmt.Println("ping数据库异常: ", err)

		return nil, err
	}

	return client.Database(tenantID), nil
}

type Collection interface {
	Clone(opts ...*options.CollectionOptions) (*mongo.Collection, error)
	Name() string
	Database() *mongo.Database
	BulkWrite(ctx context.Context, models []mongo.WriteModel,
		opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error)
	InsertOne(ctx context.Context, document interface{},
		opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error)
	InsertMany(ctx context.Context, documents []interface{},
		opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error)
	DeleteOne(ctx context.Context, filter interface{},
		opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	DeleteMany(ctx context.Context, filter interface{},
		opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	UpdateByID(ctx context.Context, id interface{}, update interface{},
		opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{},
		opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	UpdateMany(ctx context.Context, filter interface{}, update interface{},
		opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	ReplaceOne(ctx context.Context, filter interface{},
		replacement interface{}, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error)
	Aggregate(ctx context.Context, pipeline interface{},
		opts ...*options.AggregateOptions) (*mongo.Cursor, error)
	CountDocuments(ctx context.Context, filter interface{},
		opts ...*options.CountOptions) (int64, error)
	EstimatedDocumentCount(ctx context.Context,
		opts ...*options.EstimatedDocumentCountOptions) (int64, error)
	Distinct(ctx context.Context, fieldName string, filter interface{},
		opts ...*options.DistinctOptions) ([]interface{}, error)
	Find(ctx context.Context, filter interface{},
		opts ...*options.FindOptions) (*mongo.Cursor, error)
	FindOne(ctx context.Context, filter interface{},
		opts ...*options.FindOneOptions) *mongo.SingleResult
	FindOneAndDelete(ctx context.Context, filter interface{},
		opts ...*options.FindOneAndDeleteOptions) *mongo.SingleResult
	FindOneAndReplace(ctx context.Context, filter interface{},
		replacement interface{}, opts ...*options.FindOneAndReplaceOptions) *mongo.SingleResult
	FindOneAndUpdate(ctx context.Context, filter interface{},
		update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult
	Watch(ctx context.Context, pipeline interface{},
		opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error)
	Indexes() mongo.IndexView
	Drop(ctx context.Context) error
}
