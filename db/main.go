package db

import (
	"chat-system/types"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type ApplicationStorer interface {
	CreateApplication()
	GetApplication()
	UpdateApplication()
	DeleteApplication()
}

type ChatStorer interface {
	CreateChat()
	UpdateChat()
	DeleteChat()
}

type ApplicationSQLStorage struct {
	DB *sql.DB
}

func NewApplicationSQLStorage(endpoint string) (*ApplicationSQLStorage, error) {
	// what does this actually achieve ? centralized table access ? if it doesnt abstract storage then its useless
	db, err := sql.Open("mysql", endpoint)

	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	asc := &ApplicationSQLStorage{
		DB: db,
	}
	logrus.Info("Application MySQL client initialized")

	return asc, nil
}

func (asc *ApplicationSQLStorage) CreateApplication(appname string) error {
	var exists bool
	row := asc.DB.QueryRow("SELECT EXISTS(SELECT * FROM applications WHERE name=?);", appname) // sql injection ?
	if err := row.Scan(&exists); err != nil {
		fmt.Println(err.Error())
		return err
	} else if !exists {
		token := generateToken()

		if _, err := asc.DB.Exec("INSERT INTO applications (`name`, `token`, `chat_count`) VALUES (?, ?, 0)", appname, token); err != nil {
			return err
		}
		logrus.Info("Wrote application into db")
	} else if exists {
		logrus.Error("Application already exists")
		return fmt.Errorf("Application already exists")
	}
	return nil
}

func (asc *ApplicationSQLStorage) GetApplication(appname string) (*types.Application, error) {
	var app types.Application
	res, err := asc.DB.Query("SELECT * FROM applications WHERE name=?", appname)

	if err != nil {
		log.Fatal(err)
	}
	for res.Next() {
		err = res.Scan(&app.Name, &app.Token, &app.ChatCount)
		if err != nil {
			return nil, err
		}
	}
	if app == (types.Application{}) {
		return nil, fmt.Errorf("Application not found")
	}
	return &app, nil
}

//---------------------------------------------------//

type SqlStorage interface {
	Write(record map[string]string) error
	Read(idx string) (any, error)
	ReadAll() ([]any, error)
}

type KVStorage interface {
	Write(ctx context.Context, key string, val string) error
	Read(ctx context.Context, key string) (string, error)
}

type RedisStorage struct {
	client *redis.Client
}

func (rs *RedisStorage) Write(ctx context.Context, key string, val string) error {
	err := rs.client.Set(ctx, "foo", "bar", 0).Err()
	if err != nil {
		return nil
	}

	return nil
}

func (rs *RedisStorage) Read(ctx context.Context, key string) (string, error) {
	val, err := rs.client.Get(ctx, "foo").Result()
	if err != nil {
		return "", err
	}
	fmt.Println(val)
	return val, nil
}

type MySQLClient struct {
	ApplicationStorage SqlStorage
	ChatStorage        SqlStorage

	DB *sql.DB
}

func NewRedisStorage(endpoint string) *RedisStorage {
	opt, err := redis.ParseURL(endpoint)
	if err != nil {
		panic(err)
	}

	return &RedisStorage{
		client: redis.NewClient(opt),
	}
}

func NewMySQLClient(endpoint string) (*MySQLClient, error) {
	// what does this actually achieve ? centralized table access ? if it doesnt abstract storage then its useless
	db, err := sql.Open("mysql", endpoint)

	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	msc := &MySQLClient{
		DB: db,
	}

	as := ApplicationStorage{
		client: msc,
	}

	cs := ChatStorage{
		client: msc,
	}

	msc.ApplicationStorage = &as
	msc.ChatStorage = &cs
	logrus.Info("MySQL client initialized")

	return msc, nil
}

func (as *ApplicationStorage) Write(record map[string]string) error {
	var exists bool
	row := as.client.DB.QueryRow("SELECT EXISTS(SELECT * FROM applications WHERE name=?);", record["name"]) // sql injection ?
	if err := row.Scan(&exists); err != nil {
		fmt.Println(err.Error())
		return err
	} else if !exists {
		token := generateToken()

		if _, err := as.client.DB.Exec("INSERT INTO applications (`name`, `token`, `chat_count`) VALUES (?, ?, 0)", record["name"], token); err != nil {
			return err
		}
		logrus.Info("Wrote application into db")
	} else if exists {
		logrus.Error("Application already exists")
		return fmt.Errorf("Application already exists")
	}
	return nil
}

func (as *ApplicationStorage) Read(token string) (any, error) {
	var app types.Application
	res, err := as.client.DB.Query("SELECT * FROM applications WHERE name=?", token)

	if err != nil {
		log.Fatal(err)
	}
	for res.Next() {
		err = res.Scan(&app.Name, &app.Token, &app.ChatCount)
		if err != nil {
			return nil, err
		}
	}
	if app == (types.Application{}) {
		return nil, fmt.Errorf("Application not found")
	}
	return app, nil
}

func (as *ApplicationStorage) ReadAll() ([]any, error) {
	var apps []any
	res, err := as.client.DB.Query("SELECT * FROM applications")

	if err != nil {
		logrus.Error("Unable to read applications,", err)
	}

	for res.Next() {
		var app types.Application
		res.Scan(&app.Name, &app.Token, &app.ChatCount)
		apps = append(apps, app)
	}
	return apps, nil
}

func (cs *ChatStorage) Write(record map[string]string) error {
	var exists bool
	row := cs.client.DB.QueryRow("SELECT EXISTS(SELECT * FROM chats WHERE chat_number=?);", record["chatNumber"]) // sql injection ?
	if err := row.Scan(&exists); err != nil {
		fmt.Println(err.Error())
		return err
	} else if !exists {
		if _, err := cs.client.DB.Exec("INSERT INTO chats (`token`,`chat_number`, `message_count`) VALUES (?, 0, 0)", record["name"]); err != nil {
			return err
		}
		if _, err := cs.client.DB.Exec("UPDATE applications SET chat_count = chat_count + 1 WHERE name=?;)", record["name"]); err != nil {
			return err
		}
		logrus.Info("Wrote chat into db")
	} else if exists {
		logrus.Error("Chat already exists")
		return fmt.Errorf("Chat already exists")
	}
	return nil
}

func (cs *ChatStorage) Read(record map[string]string) (any, error) {
	var app types.Application
	res, err := as.client.DB.Query("SELECT * FROM applications WHERE name=?", token)

	if err != nil {
		log.Fatal(err)
	}
	for res.Next() {
		err = res.Scan(&app.Name, &app.Token, &app.ChatCount)
		if err != nil {
			return nil, err
		}
	}
	if app == (types.Application{}) {
		return nil, fmt.Errorf("Application not found")
	}
	return app, nil
}

type ApplicationStorage struct {
	client *MySQLClient
}

type ChatStorage struct {
	client *MySQLClient
}

func generateToken() string {
	rand.Seed(time.Now().UnixNano())
	randomNum := rand.Intn(math.MaxInt)
	randomNumStr := fmt.Sprintf("%d", randomNum)

	sha256Hash := sha256.New()
	sha256Hash.Write([]byte(randomNumStr))
	hashBytes := sha256Hash.Sum(nil)

	hashHex := hex.EncodeToString(hashBytes)
	randomString := hashHex[:8]
	return randomString
}
