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
	CreateApplication(string) (*types.Application, error)
	GetApplication(string) (*types.Application, error)
	GetAll() ([]any, error) // implement delete
}

type ChatStorer interface {
	CreateChat(string, int) (*types.Chat, error)
	GetChat(string, int) (*types.Chat, error)
	GetAllAppChats(string) ([]any, error)
	// DeleteChat(string) error
}

type ApplicationSQLStorage struct {
	DB *sql.DB
}

type ChatSQLStorage struct {
	DB *sql.DB
}

func NewApplicationSQLStorage(endpoint string) (*ApplicationSQLStorage, error) {
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

func NewChatSQLStorage(endpoint string) (*ChatSQLStorage, error) {
	db, err := sql.Open("mysql", endpoint)

	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	csc := &ChatSQLStorage{
		DB: db,
	}
	logrus.Info("Chat MySQL client initialized")

	return csc, nil
}

func (csc *ChatSQLStorage) GetChat(applicationToken string, chatNum int) (*types.Chat, error) {
	var chat types.Chat
	res, err := csc.DB.Query("SELECT * FROM chats WHERE token=? & chat_number", applicationToken, chatNum)

	if err != nil {
		log.Fatal(err)
	}
	for res.Next() {
		err = res.Scan(&chat.Application, &chat.Number, &chat.MessageCount)
		if err != nil {
			return nil, err
		}
	}
	if chat == (types.Chat{}) {
		return nil, fmt.Errorf("Chat not found")
	}
	return &chat, nil
}

func (csc *ChatSQLStorage) CreateChat(applicationToken string, chatNum int) (*types.Chat, error) {
	var exists bool
	var chat *types.Chat
	row := csc.DB.QueryRow("SELECT EXISTS(SELECT * FROM chats WHERE chat_number=?);", chatNum) // sql injection ?
	if err := row.Scan(&exists); err != nil {
		fmt.Println(err.Error())
		return nil, err
	} else if !exists {
		if _, err := csc.DB.Exec("INSERT INTO chats (`token`,`chat_number`, `message_count`) VALUES (?, 0, 0)", applicationToken); err != nil {
			return nil, err
		}
		if _, err := csc.DB.Exec("UPDATE applications SET chat_count = chat_count + 1 WHERE name=?;)", applicationToken); err != nil {
			return nil, err
		}
		logrus.Info("Wrote chat into db")
	} else if exists {
		logrus.Error("Chat already exists")
		return nil, fmt.Errorf("Chat already exists")
	}
	chat = types.NewChat(applicationToken, chatNum)
	return chat, nil
}

func (csc *ChatSQLStorage) GetAllAppChats(applicationToken string) ([]any, error) {
	var chats []any
	res, err := csc.DB.Query("SELECT * FROM chats WHERE token=?", applicationToken)

	if err != nil {
		logrus.Error("Unable to read chats,", err)
	}

	for res.Next() {
		var chat types.Chat
		res.Scan(&chat.Application, &chat.Number, &chat.MessageCount)
		chats = append(chats, chat)
	}
	return chats, nil
}

func (asc *ApplicationSQLStorage) CreateApplication(appname string) (*types.Application, error) {
	var exists bool
	var token string
	row := asc.DB.QueryRow("SELECT EXISTS(SELECT * FROM applications WHERE name=?);", appname) // sql injection ?
	if err := row.Scan(&exists); err != nil {
		fmt.Println(err.Error())
		return nil, err
	} else if !exists {
		token = generateToken()

		if _, err := asc.DB.Exec("INSERT INTO applications (`name`, `token`, `chat_count`) VALUES (?, ?, 0)", appname, token); err != nil {
			return nil, err
		}
		logrus.Info("Wrote application into db")
	} else if exists {
		logrus.Error("Application already exists")
		return nil, fmt.Errorf("Application already exists")
	}
	return types.NewApplication(appname, token), nil
}

func (asc *ApplicationSQLStorage) GetApplication(applicationToken string) (*types.Application, error) {
	var app types.Application
	res, err := asc.DB.Query("SELECT * FROM applications WHERE token=?", applicationToken)

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

func (asc *ApplicationSQLStorage) GetAll() ([]any, error) {
	var apps []any
	res, err := asc.DB.Query("SELECT * FROM applications")

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

func NewRedisStorage(endpoint string) *RedisStorage {
	opt, err := redis.ParseURL(endpoint)
	if err != nil {
		panic(err)
	}

	return &RedisStorage{
		client: redis.NewClient(opt),
	}
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
