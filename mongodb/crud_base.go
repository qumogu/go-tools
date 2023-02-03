package mongodb

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrorValidInserID = errors.New("invalid insertID")
	ErrNotFound       = errors.New("Data Not Found")
)

type MongoBase struct {
	C *mongo.Collection
}

func NewCrudBase(collection *mongo.Collection) *MongoBase {
	return &MongoBase{
		C: collection,
	}
}

func (b *MongoBase) Create(ctx context.Context, data interface{}) (string, error) {
	iResult, err := b.C.InsertOne(ctx, data)
	if err != nil {
		return "", err
	}
	objectID, ok := iResult.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("insertID: %v %w", objectID, ErrorValidInserID)
	}

	return objectID.Hex(), err
}

func (b *MongoBase) DeleteByIds(ctx context.Context, ids []string) (int64, error) {
	// b.C.DeleteMany()
	objIDs := b.getObjecIds(ids)

	filter := bson.M{
		"_id": bson.M{
			"$in": objIDs,
		},
	}

	result, err := b.C.DeleteMany(ctx, filter)

	return result.DeletedCount, err
}

func (b *MongoBase) UpdateByID(ctx context.Context, id string, data interface{}) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": data,
	}

	_, err = b.C.UpdateByID(ctx, objID, update)

	return err
}

// func (b *MongoBase) FindWithPage(ctx context.Context, filter interface{}, pageOrder PageOrderIntfc, results interface{}) (int64, error) {
// 	var findOpt *options.FindOptions
// 	if pageOrder != nil {
// 		findOpt = pageOrder.GetMongoFindOptions()
// 	}

// 	totalCount, err := b.C.CountDocuments(ctx, filter)
// 	if err != nil {
// 		return 0, err
// 	}

// 	cursor, err := b.C.Find(ctx, filter, findOpt)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return totalCount, cursor.All(ctx, results)
// }

func (b *MongoBase) FindWithPage(ctx context.Context, filter interface{}, pageOrder PageOrderIntfc, results interface{}) (int64, interface{}, error) {
	var findOpt *options.FindOptions
	if pageOrder != nil {
		findOpt = pageOrder.GetMongoFindOptions()
	}

	totalCount, err := b.C.CountDocuments(ctx, filter)
	if err != nil {
		return 0, nil, err
	}

	cursor, err := b.C.Find(ctx, filter, findOpt)
	if err != nil {
		return 0, nil, err
	}

	err = cursor.All(ctx, results)
	return totalCount, results, err
}

func (b *MongoBase) FindByID(ctx context.Context, id string, result interface{}) error {
	oID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": oID}
	r := b.C.FindOne(ctx, filter)
	if errors.Is(r.Err(), mongo.ErrNoDocuments) {
		return ErrNotFound
	}

	err = r.Decode(result)
	if err != nil {
		return err
	}

	return nil
}

func (b *MongoBase) getObjecIds(ids []string) []primitive.ObjectID {
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
