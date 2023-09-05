package db

import (
	"chat-system/types"
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

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
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	defer db.Close()

	if err != nil {
		return nil, err
	}
	msc := &MySQLClient{
		DB: db,
	}

	as := ApplicationStorage{
		client: msc,
	}

	msc.ApplicationStorage = &as

	return msc, nil
}

func (as *ApplicationStorage) Write(record map[string]string) error {
	var exists bool
	row := as.client.DB.QueryRow("SELECT EXISTS(SELECT * FROM applications WHERE name=?", record["name"])
	if err := row.Scan(&exists); err != nil {
		return err
	} else if !exists {
		// generate token
		if _, err := as.client.DB.Exec("INSERT INTO applications (`name`, `token`, `chat_count`) VALUES (?, ?, 0)"); err != nil {
			return err
		}
	} else if exists {
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
