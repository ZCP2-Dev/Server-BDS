package main

import (
	// 导入JSON编码/解码包
	"encoding/json"
	// 导入日志包
	"log"
	"time"

	// 导入系统信息获取库
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// 基础消息结构
// 用于封装所有API消息的通用格式
type BaseMessage struct {
	// Gateway 标识消息类型
	Gateway int `json:"gateway"`
	// Body 包含消息的具体内容
	Body json.RawMessage `json:"body"`
}

// 客户端请求结构 (C->S)
// Ping请求体
// 用于客户端发送ping请求检测服务器状态
type PingBody struct {
	// 可以为空
}

// CallApi请求体
// 用于处理客户端发送的API调用请求
type CallApiBody struct {
	// Endpoint 指定要调用的API端点
	Endpoint string `json:"endpoint"`
	// Id 请求的唯一标识符
	Id int `json:"id"`
	// Parameter 包含API调用的参数
	Parameter map[string]interface{} `json:"parameter"`
}

// 服务端响应结构 (S->C)
// Pong响应体
// 用于服务器响应ping请求
type PongBody struct {
	// ServerInfo 服务器状态信息，0为停止，1为正在开启，2为运行中
	ServerInfo int `json:"ServerInfo"`
	// CpuUsage 当前CPU使用率，为百分比
	CpuUsage int `json:"CpuUsage"`
	// MemUsage 当前内存使用率，为百分比
	MemUsage int `json:"MemUsage"`
}

// ApiReturn响应体
// 用于API调用的返回结果
type ApiReturnBody struct {
	// Id 对应请求的唯一标识符
	Id int `json:"id"`
	// Status 状态码，200表示成功
	Status int `json:"status"`
	// Info 成功时返回的数据
	Info map[string]interface{} `json:"info,omitempty"`
	// ErrInfo 错误时返回的信息
	ErrInfo string `json:"errinfo,omitempty"`
}

// API_resolve 处理接收到的API消息
// message: 接收到的原始消息字节
// clientID: 客户端唯一标识
// 返回: 处理后的响应消息字节
func API_resolve(message []byte, clientID string) []byte {
	// 解析基础消息结构
	var baseMsg BaseMessage
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		// 解析失败时记录错误日志
		log.Printf("[ERROR]解析消息失败: %v", err)
		// 返回错误响应
		// 记录错误日志
		log.Printf("[ERROR]解析消息失败，断开连接: %v", err)
		// 创建断开连接响应
		return createDisconnectResponse(0, 400, "INVALID_MESSAGE_FORMAT")
	}

	// 根据gateway值处理不同类型的请求
	switch baseMsg.Gateway {
	case 0:
		// gateway为0表示ping请求
		return handlePingRequest(clientID)
	case 1:
		// gateway为1表示CallApi请求
		var callApiBody CallApiBody
		if err := json.Unmarshal(baseMsg.Body, &callApiBody); err != nil {
			// 解析CallApi请求失败，断开连接
			log.Printf("[ERROR]解析CallApi请求失败，断开连接: %v", err)
			// 返回断开连接响应
			return createDisconnectResponse(0, 400, "INVALID_CALLAPI_FORMAT")
		}
		// 处理CallApi请求
		return handleCallApiRequest(callApiBody, clientID)
	default:
		// 未知的gateway类型，断开连接
		log.Printf("[ERROR]未知的gateway类型，断开连接: %d", baseMsg.Gateway)
		// 返回断开连接响应
		return createDisconnectResponse(0, 300, "UNKNOWN_GATEWAY")
	}
}

// 注意：clientConnections和connectionsMutex已在main.go中定义

// 处理ping请求
// clientID: 客户端唯一标识
// 返回: 包含服务器状态的pong响应或错误信息
func handlePingRequest(clientID string) []byte {
	// 检查客户端是否已登录
	connectionsMutex.RLock()
	clientConn, exists := clientConnections[clientID]
	connectionsMutex.RUnlock()

	// 对于未登录的客户端，返回错误信息
	if !exists || !clientConn.IsLoggedIn {
		log.Printf("[WARNING]未登录客户端 %s 尝试发送ping请求，已拒绝", clientID)

		// 创建错误响应
		errorResponse := BaseMessage{
			Gateway: 2, // 使用gateway 2表示错误
			Body: mustMarshal(map[string]interface{}{
				"status":  401,
				"reason":  "NOT_LOGGED_IN",
				"message": "请先登录再发送ping请求",
			}),
		}

		return mustMarshal(errorResponse)
	}

	// 获取真实系统信息
	cpuUsage, memUsage := getSystemInfo()
	// 获取服务器状态（这里假设服务器运行中，实际应用中可以根据实际情况判断）
	serverStatus := 2 // 0为停止，1为正在开启，2为运行中

	// 创建pong响应，使用真实获取的系统信息
	pongBody := PongBody{
		ServerInfo: serverStatus,
		CpuUsage:   cpuUsage,
		MemUsage:   memUsage,
	}

	// 构建完整响应
	response := BaseMessage{
		Gateway: 0, // pong的gateway为0
		Body:    mustMarshal(pongBody),
	}

	// 返回序列化后的响应
	return mustMarshal(response)
}

// 获取系统CPU和内存使用率
// 返回值: (cpu使用率百分比, 内存使用率百分比)
func getSystemInfo() (int, int) {
	// 默认返回值，防止出错
	var cpuUsage int = 0
	var memUsage int = 0

	// 使用gopsutil获取真实的CPU使用率
	// 使用百分比模式，采样间隔为0.1秒
	cpuPercentages, err := cpu.Percent(time.Millisecond*100, false)
	if err == nil && len(cpuPercentages) > 0 {
		// 将浮点数百分比转换为整数
		cpuUsage = int(cpuPercentages[0])
	} else {
		log.Printf("[WARNING]获取CPU使用率失败: %v", err)
	}

	// 使用gopsutil获取真实的内存使用率
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		// 将浮点数百分比转换为整数
		memUsage = int(memInfo.UsedPercent)
	} else {
		log.Printf("[WARNING]获取内存使用率失败: %v", err)
	}

	return cpuUsage, memUsage
}

// 处理CallApi请求
// body: 解析后的CallApi请求体
// clientID: 客户端唯一标识
// 返回: API调用结果响应
func handleCallApiRequest(body CallApiBody, clientID string) []byte {
	// 根据endpoint处理不同的API请求
	var result ApiReturnBody
	// 设置响应的请求ID
	result.Id = body.Id

	// 根据endpoint分发请求
	switch body.Endpoint {
	case "get_version":
		// 返回版本信息
		result.Status = 200
		result.Info = map[string]interface{}{
			"version":  version,  // 服务器版本
			"protocol": Protocol, // 协议版本
		}
	case "login":
		// 处理登录请求
		return HandleLoginRequest(body, clientID)
	default:
		// 未知API
		result.Status = 300
		result.ErrInfo = "API_NOT_FOUND"
	}

	// 构建完整响应
	response := BaseMessage{
		Gateway: 1, // ApiReturn的gateway为1
		Body:    mustMarshal(result),
	}

	// 返回序列化后的响应
	return mustMarshal(response)
}

// 断开连接响应体
// 用于发送断开连接通知
type DisconnectBody struct {
	// Id 对应请求的唯一标识符
	Id int `json:"id"`
	// Status 状态码
	Status int `json:"status"`
	// Reason 断开连接原因
	Reason string `json:"reason"`
}

// 创建错误响应
// id: 请求ID
// status: 状态码
// errInfo: 错误信息
// 返回: 错误响应消息
func createErrorResponse(Id int, status int, errInfo string) []byte {
	// 创建错误响应体
	result := ApiReturnBody{
		Id:      Id,
		Status:  status,
		ErrInfo: errInfo,
	}

	// 构建完整响应
	response := BaseMessage{
		Gateway: 1, // ApiReturn的gateway为1
		Body:    mustMarshal(result),
	}

	// 返回序列化后的响应
	return mustMarshal(response)
}

// 创建断开连接响应
// id: 请求ID
// status: 状态码
// reason: 断开连接原因
// 返回: 断开连接响应消息
func createDisconnectResponse(id int, status int, reason string) []byte {
	// 创建断开连接响应体
	disconnectBody := DisconnectBody{
		Id:     id,
		Status: status,
		Reason: reason,
	}

	// 构建完整响应
	response := BaseMessage{
		Gateway: 2, // 使用gateway 2表示断开连接
		Body:    mustMarshal(disconnectBody),
	}

	// 返回序列化后的响应
	return mustMarshal(response)
}

// 将go结构体序列化为JSON
// v: 要序列化的值
// 返回: 序列化后的JSON字节
func mustMarshal(v interface{}) json.RawMessage {
	// 尝试序列化
	data, err := json.Marshal(v)
	if err != nil {
		// 序列化失败时记录错误
		log.Printf("[ERROR]JSON序列化失败: %v", err)
		// 返回错误信息
		return []byte(`{"error":"JSON_SERIALIZATION_FAILED"}`)
	}
	// 返回序列化成功的数据
	return data
}
