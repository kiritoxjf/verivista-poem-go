package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strings"
)

type DB struct {
	IP     string `json:"ip_addr"`
	Port   string `json:"port"`
	Driver string `json:"driver"`
	User   string `json:"user"`
	Pass   string `json:"pass"`
	Name   string `json:"name"`
}

type Config struct {
	Token string `json:"token"`
	Time  string `json:"time"`
	DB    DB
}

type Origin struct {
	Title     string    `json:"title"`
	Dynasty   string    `json:"dynasty"`
	Author    string    `json:"author"`
	Content   []string  `json:"content"`
	Translate *[]string `json:"translate"`
}

type Data struct {
	Content string   `json:"content"`
	Origin  Origin   `json:"origin"`
	Tag     []string `json:"matchTags"`
}

type RequestBody struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

type Poem struct {
	Title     string `json:"title"`
	Dynasty   string `json:"dynasty"`
	Author    string `json:"author"`
	Content   string `json:"content"`
	All       string `json:"all"`
	Translate string `json:"translate"`
	Tag       string `json:"tag"`
}

// 数据库连接
var db *sql.DB

// 设置日志输出
func setLogOutPut() error {
	// 打开或创建日志文件
	logFilePath := os.Getenv("POEM_LOG_PATH")
	if logFilePath == "" {
		logFilePath = "logger/poem.log"
	}
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("[Open Log File Error]: %v", err)
	}
	// 设置日志输出文件
	logrus.SetOutput(file)

	return nil
}

// 获取配置信息
func getConfig() (Config, error) {
	configPath := os.Getenv("POEM_CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.json"
	}
	jsonData, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("[Config File Error or Not Exist]: %v", err)
	}

	var config Config

	err = json.Unmarshal(jsonData, &config)
	if err != nil {
		return Config{}, fmt.Errorf("[Config Analyze Error]: %v", err)
	}
	return config, nil
}

// 创建数据库连接
func setDB(DB DB) error {
	dataSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", DB.User, DB.Pass, DB.IP, DB.Port, DB.Name)

	var err error
	db, err = sql.Open(DB.Driver, dataSource)
	if err != nil {
		return fmt.Errorf("[DB Connect Create Error]: %v", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("[DB Ping Fail]: %v", err)
	}

	return nil
}

// 获取诗词
func getPoem(token string) error {
	// 发送请求
	url := "https://v2.jinrishici.com/sentence"

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("[Get Poem Http Request Create Error]：%v", err)
	}

	req.Header.Set("X-User-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("[Get Poem Http Request Error]: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logrus.Errorln("[Poem Body Close Error]: %v", err)
		}
	}(resp.Body)

	logrus.Infoln("[Get Poem Status]: ", resp.Status)
	// 解析Body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("[Error Reading Poem Body]: %v", err)
	}

	var requestBody RequestBody

	err = json.Unmarshal(body, &requestBody)
	if err != nil {
		return fmt.Errorf("[Error Analyze Poem Body]: %v", err)
	}

	poem := Poem{
		Title:   requestBody.Data.Origin.Title,
		Dynasty: requestBody.Data.Origin.Dynasty,
		Author:  requestBody.Data.Origin.Author,
		Content: requestBody.Data.Content,
		All:     strings.Join(requestBody.Data.Origin.Content, ""),
		Tag:     strings.Join(requestBody.Data.Tag, ", "),
	}

	if requestBody.Data.Origin.Translate == nil {
		poem.Translate = ""
	} else {
		poem.Translate = strings.Join(*requestBody.Data.Origin.Translate, "")
	}

	err = storePoem(poem)
	if err != nil {
		return err
	}

	return nil
}

// 写入数据库
func storePoem(poem Poem) error {
	stmt, err := db.Prepare("INSERT INTO t_poem SET title=?, dynasty=?, author=?, content=?, `all`=?, `translate`=?, tag=?")
	if err != nil {
		return fmt.Errorf("[Error preparing SQL statement]: %v", err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			logrus.Errorln("[Error DB preparing Closing]:", err)
		}
	}(stmt)

	_, err = stmt.Exec(poem.Title, poem.Dynasty, poem.Author, poem.Content, poem.All, poem.Translate, poem.Tag)
	if err != nil {
		return fmt.Errorf("[Error executing SQL statement]: %v", err)
	}
	logrus.Infoln("Success Get Poem")
	return nil
}

func main() {
	err := setLogOutPut()
	if err != nil {
		logrus.Errorln(err)
		return
	}

	config, err := getConfig()
	if err != nil {
		logrus.Errorln(err)
		return
	}

	err = setDB(config.DB)
	if err != nil {
		logrus.Errorln(err)
		return
	}

	// 创建一个新的 Cron 实例
	c := cron.New()
	// 每天双数小时点获取新的诗写入数据库
	err = c.AddFunc(config.Time, func() {
		logrus.Infoln("Start Get Poem!")
		err = getPoem(config.Token)
		if err != nil {
			logrus.Errorln(err)
			return
		}
		// 在这里编写每日任务的具体逻辑
	})
	if err != nil {
		logrus.Errorln("[Time Func Create Error]: ", err)
		return
	}

	// 启动 Cron
	c.Start()
	logrus.Infoln("Poem EXE START;")

	// 阻塞主 goroutine，以保持程序运行
	select {}
}
