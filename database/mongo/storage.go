package mongo

import (
	"errors"
	"time"

	"github.com/staticbackendhq/core/internal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LocalFile struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	AccountID primitive.ObjectID `bson:"accountId" json:"accountId"`
	Key       string             `bson:"key" json:"key"`
	URL       string             `bson:"url" json:"url"`
	Size      int64              `bson:"size" json:"size"`
	Uploaded  time.Time          `bson:"on" json:"uploaded"`
}

func toLocalFile(f internal.File) LocalFile {
	id, err := primitive.ObjectIDFromHex(f.ID)
	if err != nil {
		return LocalFile{}
	}

	acctID, err := primitive.ObjectIDFromHex(f.AccountID)
	if err != nil {
		return LocalFile{}
	}

	return LocalFile{
		ID:        id,
		AccountID: acctID,
		Key:       f.Key,
		URL:       f.URL,
		Size:      f.Size,
		Uploaded:  f.Uploaded,
	}
}

func fromLocalFile(lf LocalFile) internal.File {
	return internal.File{
		ID:        lf.ID.Hex(),
		AccountID: lf.AccountID.Hex(),
		Key:       lf.Key,
		URL:       lf.URL,
		Size:      lf.Size,
		Uploaded:  lf.Uploaded,
	}
}

func (mg *Mongo) AddFile(dbName string, f internal.File) (id string, err error) {
	db := mg.Client.Database(dbName)

	f.ID = primitive.NewObjectID().Hex()

	lf := toLocalFile(f)

	res, err := db.Collection("sb_files").InsertOne(mg.Ctx, lf)
	if err != nil {
		return
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return id, errors.New("unable to get inserted id for file")
	}

	id = oid.Hex()
	return
}

func (mg *Mongo) GetFileByID(dbName, fileID string) (f internal.File, err error) {
	db := mg.Client.Database(dbName)

	oid, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return
	}

	var result LocalFile

	filter := bson.M{FieldID: oid}

	sr := db.Collection("sb_files").FindOne(mg.Ctx, filter)
	if err = sr.Decode(&result); err != nil {
		return
	} else if err = sr.Err(); err != nil {
		return
	}

	f = fromLocalFile(result)
	return
}

func (mg *Mongo) DeleteFile(dbName, fileID string) error {
	db := mg.Client.Database(dbName)

	oid, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return err
	}

	filter := bson.M{FieldID: oid}
	if _, err := db.Collection("sb_files").DeleteOne(mg.Ctx, filter); err != nil {
		return err
	}
	return nil
}
