package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BaseInterfc interface {
	Create(ctx context.Context, data interface{}) (string, error)
	CreateMany(ctx context.Context, documents []interface{}) ([]string, error)       // 批量创建, 返回id列表
	DeleteFalseByIds(ctx context.Context, ids []string) (int64, error)               // 假删除
	DeleteByIds(ctx context.Context, ids []string) (int64, error)                    // 真删除
	DeleteByIdsOrgId(ctx context.Context, ids []string, orgId string) (int64, error) // 真删除
	DeleteByFilter(ctx context.Context, filter interface{}) (int64, error)
	UpdateByID(ctx context.Context, id string, data interface{}) error // 根据主键ID更新记录
	UpdateOneByFilter(ctx context.Context, filter interface{}, data interface{}) error
	UpdateMany(ctx context.Context, filter interface{}, data interface{}) error
	ReplaceOneById(ctx context.Context, id string, data interface{}) error
	Upsert(ctx context.Context, filter interface{}, data interface{}) (string, error)
	Find(ctx context.Context, filter interface{}, opt PageOrderIntfc, results interface{}) error
	FindWithPage(ctx context.Context, filter interface{}, pageOrder PageOrderIntfc, results interface{}) (int64, error)
	FindByNameWithPage(ctx context.Context, name string, pageOrder PageOrderIntfc, results interface{}) (int64, error)
	FindOne(ctx context.Context, filter interface{}, result interface{}) error
	FindOneByID(ctx context.Context, id string, result interface{}) error
	IsExist(ctx context.Context, filter interface{}) (bool, error)
	IsExistByName(ctx context.Context, name string) (bool, error)
	Count(ctx context.Context, filter interface{}) (int64, error)
}

type Base struct {
	collection Collection
}

func NewBase(collection Collection) *Base {
	return &Base{collection}
}

func (b *Base) Create(ctx context.Context, data interface{}) (string, error) {
	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	mongoResult, err := b.collection.InsertOne(ctx, data)
	if err != nil {
		return "", err
	}

	objID, ok := mongoResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", errors.New("id error")
	}

	return objID.Hex(), nil
}

// 假删除.
func (b *Base) DeleteFalseByIds(ctx context.Context, ids []string) (int64, error) {
	objIDs := b.getObjecIds(ids)

	filter := bson.M{
		"_id": bson.M{
			"$in": objIDs,
		},
	}

	update := bson.M{
		"$set": bson.M{
			"is_delete": true,
		},
	}

	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	result, err := b.collection.UpdateMany(ctx, filter, update)

	return result.MatchedCount, err
}

// 真删除.
func (b *Base) DeleteByIds(ctx context.Context, ids []string) (int64, error) {
	objIDs := b.getObjecIds(ids)

	filter := bson.M{
		"_id": bson.M{
			"$in": objIDs,
		},
	}

	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	result, err := b.collection.DeleteMany(ctx, filter)

	return result.DeletedCount, err
}

// 真删除.
func (b *Base) DeleteByIdsOrgId(ctx context.Context, ids []string, orgId string) (int64, error) {
	objIDs := b.getObjecIds(ids)

	filter := bson.M{
		"_id": bson.M{
			"$in": objIDs,
		},
	}

	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	result, err := b.collection.DeleteMany(ctx, filter)

	return result.DeletedCount, err
}

func (b *Base) DeleteByFilter(ctx context.Context, filter interface{}) (int64, error) {
	result, err := b.collection.DeleteMany(ctx, filter)

	return result.DeletedCount, err
}

// update by id
//
// data 可以为结构体model，支持 bson tag
//
// data 可以为 map[string]interface{}或bson.M.
func (b *Base) UpdateByID(ctx context.Context, id string, data interface{}) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	update := bson.M{
		"$set": data,
	}
	_, err = b.collection.UpdateByID(ctx, objID, update)

	return err
}

func (b *Base) UpdateOneByFilter(ctx context.Context, filter interface{}, data interface{}) error {
	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	update := bson.M{
		"$set": data,
	}

	// _, err = b.collection.UpdateByID(ctx, objID, update)
	_, err := b.collection.UpdateOne(ctx, filter, update, nil)

	return err
}

func (b *Base) UpdateMany(ctx context.Context, filter interface{}, data interface{}) error {
	_, err := b.collection.UpdateMany(ctx, filter, data, nil)

	return err
}

// 替换文档,也相当于更新文档
//
// data为model.
func (b *Base) ReplaceOneById(ctx context.Context, id string, data interface{}) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id": objID,
	}

	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	_, err = b.collection.ReplaceOne(ctx, filter, data)

	return err
}

// 存在则更新， 不存在则添加.
func (b *Base) Upsert(ctx context.Context, filter interface{}, data interface{}) (string, error) {
	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	opt := options.Replace().SetUpsert(true)

	mongoResult, err := b.collection.ReplaceOne(ctx, filter, data, opt)
	if err != nil {
		return "", err
	}

	objID, ok := mongoResult.UpsertedID.(primitive.ObjectID)
	if !ok {
		return "", errors.New("id error")
	}

	return objID.Hex(), nil
}

// 查询过滤, 返回所有结果，不分页.
func (b *Base) Find(ctx context.Context, filter interface{}, opt PageOrderIntfc, results interface{}) error {
	var findOpt *options.FindOptions
	if opt != nil {
		findOpt = opt.GetMongoFindOptions()
	}

	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	cursor, err := b.collection.Find(ctx, filter, findOpt)
	if err != nil {
		return err
	}

	return cursor.All(ctx, results)
}

// 分页查询， 返回总记录个数.
// results 为切片指针.
func (b *Base) FindWithPage(ctx context.Context, filter interface{}, pageOrder PageOrderIntfc, results interface{}) (int64, error) {
	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	var findOpt *options.FindOptions
	if pageOrder != nil {
		findOpt = pageOrder.GetMongoFindOptions()
	}

	totalCount, err := b.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	cursor, err := b.collection.Find(ctx, filter, findOpt)
	if err != nil {
		return 0, err
	}

	// m := make([]map[string]interface{}, 0, 16)
	// err = cursor.All(ctx, &m)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("%#v\n", m)
	return totalCount, cursor.All(ctx, results)
}

// 通过Name过滤查询，进行分页.
func (b *Base) FindByNameWithPage(ctx context.Context, name string, pageOrder PageOrderIntfc, results interface{}) (int64, error) {
	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	filter := bson.M{}

	if name != "" {
		filter["name"] = name
	}

	return b.FindWithPage(ctx, filter, pageOrder, results)
}

// 返回第一条结果.
func (b *Base) FindOne(ctx context.Context, filter interface{}, result interface{}) error {
	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	mongoResult := b.collection.FindOne(ctx, filter)
	if mongoResult.Err() != nil {
		if errors.Is(mongoResult.Err(), mongo.ErrNoDocuments) {
			return ErrNotFound
		}

		return mongoResult.Err()
	}

	return mongoResult.Decode(result)
}

func (b *Base) FindOneByID(ctx context.Context, id string, result interface{}) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{
		"_id": objID,
	}

	return b.FindOne(ctx, filter, result)
}

// 是否存在.
func (b *Base) IsExist(ctx context.Context, filter interface{}) (bool, error) {
	ctx, cancel := NewContextWithTimeout(ctx)
	defer cancel()

	totalCount, err := b.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return totalCount > 0, err
}

func (b *Base) IsExistByName(ctx context.Context, name string) (bool, error) {
	filter := bson.M{
		"name": name,
	}

	return b.IsExist(ctx, filter)
}

func (b *Base) Count(ctx context.Context, filter interface{}) (int64, error) {
	return b.collection.CountDocuments(ctx, filter)
}

func (b *Base) getObjecIds(ids []string) []primitive.ObjectID {
	objIDs := make([]primitive.ObjectID, 0, len(ids))

	for _, id := range ids {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}

		objIDs = append(objIDs, objID)
	}

	return objIDs
}

// 批量创建, 返回  primitive.ObjectID 列表.
func (b *Base) CreateMany(ctx context.Context, documents []interface{}) ([]string, error) {
	result, err := b.collection.InsertMany(ctx, documents)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(result.InsertedIDs))

	for _, id := range result.InsertedIDs {
		objID, ok := id.(primitive.ObjectID)
		if ok {
			ids = append(ids, objID.Hex())
		}
	}

	return ids, err
}

const ( // timeout.
	DefaultTimeout = time.Second * 10
)

func NewContextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, DefaultTimeout)
}
