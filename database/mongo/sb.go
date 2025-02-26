package mongo

import (
	"time"

	"github.com/staticbackendhq/core/internal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LocalCustomer struct {
	ID             primitive.ObjectID `bson:"_id" json:"id"`
	Email          string             `bson:"email" json:"email"`
	StripeID       string             `bson:"stripeId" json:"stripeId"`
	SubscriptionID string             `bson:"subId" json:"subId"`
	IsActive       bool               `bson:"active" json:"-"`
	Created        time.Time          `bson:"created" json:"created"`
}

func toLocalCustomer(c internal.Customer) LocalCustomer {
	return LocalCustomer{
		Email:          c.Email,
		StripeID:       c.StripeID,
		SubscriptionID: c.SubscriptionID,
		IsActive:       c.IsActive,
		Created:        c.Created,
	}
}

func fromLocalCustomer(c LocalCustomer) internal.Customer {
	return internal.Customer{
		ID:             c.ID.Hex(),
		Email:          c.Email,
		StripeID:       c.StripeID,
		SubscriptionID: c.SubscriptionID,
		IsActive:       c.IsActive,
		Created:        c.Created,
	}
}

func (mg *Mongo) CreateCustomer(customer internal.Customer) (internal.Customer, error) {
	db := mg.Client.Database("sbsys")

	lc := toLocalCustomer(customer)
	lc.ID = primitive.NewObjectID()

	if _, err := db.Collection("accounts").InsertOne(mg.Ctx, lc); err != nil {
		return customer, err
	}
	return fromLocalCustomer(lc), nil
}

type LocalBase struct {
	ID               primitive.ObjectID `bson:"_id" json:"id"`
	SBID             primitive.ObjectID `bson:"accountId" json:"-"`
	Name             string             `bson:"name" json:"name"`
	Whitelist        []string           `bson:"whitelist" json:"whitelist"`
	IsActive         bool               `bson:"active" json:"-"`
	MonthlyEmailSent int                `bson:"mes" json:"-"`
}

func toLocalBase(b internal.BaseConfig) LocalBase {
	id, err := primitive.ObjectIDFromHex(b.CustomerID)
	if err != nil {
		return LocalBase{}
	}

	return LocalBase{
		SBID:             id,
		Name:             b.Name,
		Whitelist:        b.AllowedDomain,
		IsActive:         b.IsActive,
		MonthlyEmailSent: b.MonthlySentEmail,
	}
}

func fromLocalBase(b LocalBase) internal.BaseConfig {
	return internal.BaseConfig{
		ID:               b.ID.Hex(),
		CustomerID:       b.SBID.Hex(),
		Name:             b.Name,
		AllowedDomain:    b.Whitelist,
		IsActive:         b.IsActive,
		MonthlySentEmail: b.MonthlyEmailSent,
	}
}

func (mg *Mongo) CreateBase(base internal.BaseConfig) (internal.BaseConfig, error) {
	db := mg.Client.Database("sbsys")

	lb := toLocalBase(base)
	lb.ID = primitive.NewObjectID()

	if _, err := db.Collection("bases").InsertOne(mg.Ctx, lb); err != nil {
		return base, err
	}
	return fromLocalBase(lb), nil
}

func (mg *Mongo) EmailExists(email string) (bool, error) {
	db := mg.Client.Database("sbsys")

	count, err := db.Collection("accounts").CountDocuments(mg.Ctx, bson.M{"email": email})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (mg *Mongo) FindAccount(customerID string) (cus internal.Customer, err error) {
	db := mg.Client.Database("sbsys")

	accountID, err := primitive.ObjectIDFromHex(customerID)
	if err != nil {
		return
	}

	var lc LocalCustomer

	filter := bson.M{FieldID: accountID}
	sr := db.Collection("accounts").FindOne(mg.Ctx, filter)
	err = sr.Decode(&lc)
	cus = fromLocalCustomer(lc)
	return
}

func (mg *Mongo) FindDatabase(baseID string) (conf internal.BaseConfig, err error) {
	db := mg.Client.Database("sbsys")

	id, err := primitive.ObjectIDFromHex(baseID)
	if err != nil {
		return
	}

	var lb LocalBase
	sr := db.Collection("bases").FindOne(mg.Ctx, bson.M{FieldID: id})
	err = sr.Decode(&lb)
	conf = fromLocalBase(lb)
	return
}

func (mg *Mongo) DatabaseExists(name string) (bool, error) {
	db := mg.Client.Database("sbsys")

	count, err := db.Collection("bases").CountDocuments(mg.Ctx, bson.M{"name": name})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (mg *Mongo) ListDatabases() (results []internal.BaseConfig, err error) {
	db := mg.Client.Database("sbsys")

	filter := bson.M{FieldIsActive: true}

	cur, err := db.Collection("bases").Find(mg.Ctx, filter)
	if err != nil {
		return
	}
	defer cur.Close(mg.Ctx)

	for cur.Next(mg.Ctx) {
		var lb LocalBase
		if err = cur.Decode(&lb); err != nil {
			return
		}

		results = append(results, fromLocalBase(lb))
	}
	if err = cur.Err(); err != nil {
		return
	}

	return
}

func (mg *Mongo) GetCustomerByStripeID(stripeID string) (cus internal.Customer, err error) {
	db := mg.Client.Database("sbsys")

	var acct LocalCustomer
	sr := db.Collection("accounts").FindOne(mg.Ctx, bson.M{"stripeId": stripeID})
	if err = sr.Decode(&acct); err != nil {
		return
	} else if err = sr.Err(); err != nil {
		return
	}

	cus = fromLocalCustomer(acct)
	return
}

func (mg *Mongo) IncrementMonthlyEmailSent(baseID string) error {
	db := mg.Client.Database("sbsys")

	id, err := primitive.ObjectIDFromHex(baseID)
	if err != nil {
		return err
	}

	filter := bson.M{FieldID: id}
	update := bson.M{"$inc": bson.M{"mes": 1}}
	if _, err := db.Collection("bases").UpdateOne(mg.Ctx, filter, update); err != nil {
		return err
	}
	return nil
}

func (mg *Mongo) ActivateCustomer(customerID string) error {
	db := mg.Client.Database("sbsys")

	oid, err := primitive.ObjectIDFromHex(customerID)
	if err != nil {
		return err
	}

	filter := bson.M{FieldID: oid}
	update := bson.M{"$set": bson.M{"active": true}}

	res := db.Collection("accounts").FindOneAndUpdate(mg.Ctx, filter, update)
	if err := res.Err(); err != nil {
		return err
	}

	filter = bson.M{FieldAccountID: oid}
	res = db.Collection("bases").FindOneAndUpdate(mg.Ctx, filter, update)
	return res.Err()
}

func (mg *Mongo) NewID() string {
	return primitive.NewObjectID().Hex()
}

func (mg *Mongo) DeleteCustomer(dbName, email string) error {
	db := mg.Client.Database(dbName)

	if err := db.Drop(mg.Ctx); err != nil {
		return err
	}

	db = mg.Client.Database("sbsys")

	filter := bson.M{"email": email}
	if _, err := db.Collection("accounts").DeleteMany(mg.Ctx, filter); err != nil {
		return err
	}

	filter = bson.M{"name": dbName}
	if _, err := db.Collection("bases").DeleteMany(mg.Ctx, filter); err != nil {
		return err
	}

	return nil
}
