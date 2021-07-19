package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"github.com/gbrlsnchs/jwt/v3"
)

type Account struct {
	ID    primitive.ObjectID `bson:"_id" json:"id"`
	Email string             `bson:"email" json:"email"`
}

type Token struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	AccountID primitive.ObjectID `bson:"accountId" json:"accountId"`
	Token     string             `bson:"token" json:"token"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"pw" json:"-"`
	Role      int                `bson:"role" json:"role"`
}

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func emailExists(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(r.URL.Query().Get("e"))
	if len(email) == 0 {
		respond(w, http.StatusOK, false)
		return
	}

	conf, ok := r.Context().Value(ContextBase).(BaseConfig)
	if !ok {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	db := client.Database(conf.Name)

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	count, err := db.Collection("sb_tokens").CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, count == 1)
}

func login(w http.ResponseWriter, r *http.Request) {
	conf, ok := r.Context().Value(ContextBase).(BaseConfig)
	if !ok {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	db := client.Database(conf.Name)

	var l Login
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	l.Email = strings.ToLower(l.Email)

	tok, err := validateUserPassword(db, l.Email, l.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token := fmt.Sprintf("%s|%s", tok.ID.Hex(), tok.Token)

	// get their JWT
	jwtBytes, err := getJWT(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tokens[token] = Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      tok.Role,
	}

	respond(w, http.StatusOK, string(jwtBytes))
}

func validateUserPassword(db *mongo.Database, email, password string) (*Token, error) {
	email = strings.ToLower(email)

	ctx := context.Background()
	sr := db.Collection("sb_tokens").FindOne(ctx, bson.M{"email": email})

	var tok Token
	if err := sr.Decode(&tok); err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(tok.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid email/password")
	}

	return &tok, nil
}

func register(w http.ResponseWriter, r *http.Request) {
	conf, ok := r.Context().Value(ContextBase).(BaseConfig)
	if !ok {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		log.Println("invalid StaticBackend key")
		return
	}

	db := client.Database(conf.Name)

	var l Login
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	l.Email = strings.ToLower(l.Email)

	// make sure this email does not exists
	count, err := db.Collection("sb_tokens").CountDocuments(context.Background(), bson.M{"email": l.Email})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if count > 0 {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}

	jwtBytes, _, err := createAccountAndUser(db, l.Email, l.Password, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, string(jwtBytes))
}

func createAccountAndUser(db *mongo.Database, email, password string, role int) ([]byte, Token, error) {
	acctID := primitive.NewObjectID()

	a := Account{
		ID:    acctID,
		Email: email,
	}

	ctx := context.Background()
	_, err := db.Collection("sb_accounts").InsertOne(ctx, a)
	if err != nil {
		return nil, Token{}, err
	}

	jwtBytes, tok, err := createUser(db, acctID, email, password, role)
	if err != nil {
		return nil, Token{}, err
	}
	return jwtBytes, tok, nil
}

func createUser(db *mongo.Database, accountID primitive.ObjectID, email, password string, role int) ([]byte, Token, error) {
	ctx := context.Background()

	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, Token{}, err
	}

	tok := Token{
		ID:        primitive.NewObjectID(),
		AccountID: accountID,
		Email:     email,
		Token:     primitive.NewObjectID().Hex(),
		Password:  string(b),
		Role:      role,
	}

	_, err = db.Collection("sb_tokens").InsertOne(ctx, tok)
	if err != nil {
		return nil, tok, err
	}

	token := fmt.Sprintf("%s|%s", tok.ID.Hex(), tok.Token)

	// Get their JWT
	jwtBytes, err := getJWT(token)
	if err != nil {
		return nil, tok, err
	}

	tokens[token] = Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      role,
	}

	return jwtBytes, tok, nil
}

func setRole(w http.ResponseWriter, r *http.Request) {
	a, ok := r.Context().Value(ContextAuth).(Auth)
	if !ok || a.Role < 100 {
		http.Error(w, "insufficient priviledges", http.StatusUnauthorized)
		return
	}

	conf, ok := r.Context().Value(ContextBase).(BaseConfig)
	if !ok {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		log.Println("invalid StaticBackend key")
		return
	}

	var data = new(struct {
		Email string `json:"email"`
		Role  int    `json:"role"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := client.Database(conf.Name)

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	filter := bson.M{"email": data.Email}
	update := bson.M{"$set": bson.M{"role": data.Role}}
	if _, err := db.Collection("sb_tokens").UpdateOne(ctx, filter, update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func setPassword(w http.ResponseWriter, r *http.Request) {
	a, ok := r.Context().Value(ContextAuth).(Auth)
	if !ok || a.Role < 100 {
		http.Error(w, "insufficient priviledges", http.StatusUnauthorized)
		return
	}

	conf, ok := r.Context().Value(ContextBase).(BaseConfig)
	if !ok {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		log.Println("invalid StaticBackend key")
		return
	}

	var data = new(struct {
		Email       string `json:"email"`
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := client.Database(conf.Name)

	tok, err := validateUserPassword(db, data.Email, data.OldPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	newpw, err := bcrypt.GenerateFromPassword([]byte(data.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	filter := bson.M{"_id": tok.ID}
	update := bson.M{"$set": bson.M{"pw": string(newpw)}}
	if _, err := db.Collection("sb_tokens").UpdateOne(ctx, filter, update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func resetPassword(w http.ResponseWriter, r *http.Request) {
	conf, ok := r.Context().Value(ContextBase).(BaseConfig)
	if !ok {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	db := client.Database(conf.Name)

	var data = new(struct {
		Email string `json:"email"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.M{"email": strings.ToLower(data.Email)}
	count, err := db.Collection("sb_tokens").CountDocuments(context.Background(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if count == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	code := randStringRunes(6)
	update := bson.M{"%set": bson.M{"sb_reset_code": code}}
	if _, err := db.Collection("sb_tokens").UpdateOne(context.Background(), filter, update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body := fmt.Sprintf(`Your reset code is: %s`, code)
	if err := sendMail(data.Email, "", FromEmail, FromName, "Your password reset code", body, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func changePassword(w http.ResponseWriter, r *http.Request) {
	conf, ok := r.Context().Value(ContextBase).(BaseConfig)
	if !ok {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	db := client.Database(conf.Name)

	var data = new(struct {
		Email    string `json:"email"`
		Code     string `json:"code"`
		Password string `json:"password"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.M{"email": strings.ToLower(data.Email), "sb_reset_code": data.Code}
	var tok Token
	sr := db.Collection("sb_tokens").FindOne(context.Background(), filter)
	if err := sr.Decode(&tok); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newpw, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	filter = bson.M{fieldID: tok.ID}
	update := bson.M{"$set": bson.M{"pw": string(newpw)}}
	if _, err := db.Collection("sb_tokens").UpdateOne(ctx, filter, update); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func getJWT(token string) ([]byte, error) {
	now := time.Now()
	pl := JWTPayload{
		Payload: jwt.Payload{
			Issuer:         "StaticBackend",
			ExpirationTime: jwt.NumericDate(now.Add(12 * time.Hour)),
			NotBefore:      jwt.NumericDate(now.Add(30 * time.Minute)),
			IssuedAt:       jwt.NumericDate(now),
			JWTID:          primitive.NewObjectID().Hex(),
		},
		Token: token,
	}

	return jwt.Sign(pl, hs)

}

func sudoGetTokenFromAccountID(w http.ResponseWriter, r *http.Request) {
	conf, _, err := extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := client.Database(conf.Name)

	id := ""

	_, r.URL.Path = ShiftPath(r.URL.Path)
	id, r.URL.Path = ShiftPath(r.URL.Path)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.M{fieldAccountID: oid}
	ctx := context.Background()

	opt := options.Find()
	opt.SetLimit(1)
	opt.SetSort(bson.M{fieldID: 1})

	cur, err := db.Collection("sb_tokens").Find(ctx, filter, opt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)

	var tok Token
	if cur.Next(ctx) {
		if err := cur.Decode(&tok); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if len(tok.Token) == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	token := fmt.Sprintf("%s|%s", tok.ID.Hex(), tok.Token)

	jwtBytes, err := getJWT(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tokens[token] = Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      tok.Role,
	}

	respond(w, http.StatusOK, string(jwtBytes))
}
