package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"sync"
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

// 用于存储客户端连接状态
var clientConnections = make(map[string]*ClientConnection)
var connectionsMutex sync.RWMutex

// ClientConnection 表示一个客户端连接
type ClientConnection struct {
	Conn         *websocket.Conn
	IsLoggedIn   bool
	LastPingTime time.Time
	RemoteAddr   string
}

// SetClientLoggedIn 设置客户端的登录状态
// clientID: 客户端唯一标识
// isLoggedIn: 登录状态
func SetClientLoggedIn(clientID string, isLoggedIn bool) {
	connectionsMutex.Lock()
	if clientConn, exists := clientConnections[clientID]; exists {
		clientConn.IsLoggedIn = isLoggedIn
		log.Printf("[INFO]客户端 %s 登录状态已更新为: %v", clientID, isLoggedIn)
	}
	connectionsMutex.Unlock()
}

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

		// 获取客户端远程地址作为标识
		clientID := r.RemoteAddr
		log.Printf("[INFO]客户端已连接: %s", clientID)

		// 创建客户端连接对象并存储
		clientConn := &ClientConnection{
			Conn:         conn,
			IsLoggedIn:   false,
			LastPingTime: time.Now(),
			RemoteAddr:   clientID,
		}

		connectionsMutex.Lock()
		clientConnections[clientID] = clientConn
		connectionsMutex.Unlock()

		// 确保在函数退出时删除连接信息
		defer func() {
			connectionsMutex.Lock()
			delete(clientConnections, clientID)
			connectionsMutex.Unlock()
			log.Printf("[INFO]客户端已断开连接: %s", clientID)
		}()

		// 设置ping处理器来检测连接活跃状态
		conn.SetPongHandler(func(string) error {
			connectionsMutex.Lock()
			if connInfo, exists := clientConnections[clientID]; exists {
				connInfo.LastPingTime = time.Now()
			}
			connectionsMutex.Unlock()
			return nil
		})

		// 启动ping发送协程，每分钟检查连接状态
		pingTicker := time.NewTicker(time.Minute)
		defer pingTicker.Stop()

		go func() {
			for range pingTicker.C {
				connectionsMutex.RLock()
				connInfo, exists := clientConnections[clientID]
				connectionsMutex.RUnlock()

				if !exists {
					return
				}

				// 检查是否超过一分钟未收到ping包
				if time.Since(connInfo.LastPingTime) > time.Minute {
					log.Printf("[WARNING]客户端 %s 超过一分钟未收到ping包，断开连接", clientID)
					conn.Close()
					return
				}

				// 发送ping包以保持连接活跃
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err != nil {
					log.Printf("[ERROR]发送ping消息失败: %v", err)
					return
				}
			}
		}()

		// 持续监听消息
		for {
			// 读取消息
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[ERROR]读取消息错误: %v", err)
				break
			}

			//解析消息
			responseData := API_resolve(message, clientID) //进入API处理并获取响应

			// 如果有响应数据，则发送
			if len(responseData) > 0 {
				err = conn.WriteMessage(websocket.TextMessage, responseData)
				if err != nil {
					log.Printf("[ERROR]发送消息错误: %v", err)
					break
				}
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
