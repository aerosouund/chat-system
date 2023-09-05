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
	// takes some table as input to dictate methods
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

	msc.ApplicationStorage = &as
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
	res, err := as.client.DB.Query("SELECT * FROM applications WHERE token=?")

	if err != nil {
		log.Fatal(err)
	}

	err = res.Scan(&app.Name, &app.Token, &app.ChatCount)
	if err != nil {
		return nil, err
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

type ApplicationStorage struct {
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
