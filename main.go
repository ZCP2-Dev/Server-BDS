package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
)

// 配置结构体
type Config struct {
	Port string `json:"port"`
}

// 加载配置文件
func loadConfig(path string) (*Config, error) {
	// 读取文件内容
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 解析JSON
	var config Config
	jsonErr := json.Unmarshal(data, &config)
	if jsonErr != nil {
		return nil, jsonErr
	}

	return &config, nil
}

var version = "DEV20250816" //全局 版本号
var Protocol = "500"        //全局 协议版本号

// 启动WebSocket服务器
func OpenWebsocket(config *Config) {
	// 获取端口号
	port := config.Port
	if port == "" {
		port = ":62001" // 默认端口
	}

	// 设置WebSocket升级器
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// 允许所有CORS请求，生产环境应限制来源
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// 定义WebSocket处理函数
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// 将HTTP连接升级为WebSocket连接
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ERROR]WebSocket升级失败: %v", err)
			return
		}
		defer conn.Close()

		log.Printf("[INFO]客户端已连接: %s", r.RemoteAddr)

		// 持续监听消息
		for {
			// 读取消息
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[ERROR]读取消息错误: %v", err)
				break
			}

			//解析消息
			API_resolve(message) //进入API处理

			// 回复消息
			err = conn.WriteMessage(websocket.TextMessage, responseData)
			if err != nil {
				log.Printf("[ERROR]发送消息错误: %v", err)
				break
			}
		}
	})

	// 启动HTTP服务器
	log.Printf("[INFO]WebSocket服务器已启动，监听端口: %s", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("[ERROR]服务器启动失败: %v", err)
	}
}

func main() {
	// 开始运行该程序的提示
	log.Printf("[INFO]ZCP2-Server-BDS已启动，版本: %s，协议版本:%s", version, Protocol)
	startTime := time.Now()
	log.Printf("[INFO]启动时间: %s", startTime.Format("2006-01-02 15:04:05"))

	// 读取配置文件
	configPath := filepath.Join("Panel_Setting", "config.json")
	config, err := loadConfig(configPath)
	if err != nil {
		log.Printf("[ERROR]读取配置文件失败，使用默认端口: %v", err)
		// 使用默认端口
		config = &Config{
			Port: ":62001",
		}
	}
	// 启动websocket连接
	OpenWebsocket(config)
}
