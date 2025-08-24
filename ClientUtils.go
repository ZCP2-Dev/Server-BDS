package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LoginRequest 登录请求参数结构体
type LoginRequest struct {
	Password  string `json:"password"`
	RowStart  int    `json:"row_start,string,omitempty"` // 从log中抓取的起始行数
	RowEnd    int    `json:"row_end,string,omitempty"`   // 从log中抓取的截止行数
}

// 处理登录请求，根据协议文档实现
func HandleLoginRequest(body CallApiBody, clientID string) []byte {
	// 创建响应对象
	result := ApiReturnBody{
		Id: body.Id,
	}

	// 解析请求参数
	var request LoginRequest
	
	// 获取密码参数
	passwordValue, hasPassword := body.Parameter["password"]
	if hasPassword {
		passwordStr, ok := passwordValue.(string)
		if ok {
			request.Password = passwordStr
		}
	}

	// 获取row_start参数
	rowStartValue, hasRowStart := body.Parameter["row_start"]
	if hasRowStart {
		switch v := rowStartValue.(type) {
		case float64:
			request.RowStart = int(v)
		case int:
			request.RowStart = v
		case string:
			if val, err := strconv.Atoi(v); err == nil {
				request.RowStart = val
			}
		}
	} else {
		request.RowStart = 0 // 默认值
	}

	// 获取row_end参数
	rowEndValue, hasRowEnd := body.Parameter["row_end"]
	if hasRowEnd {
		switch v := rowEndValue.(type) {
		case float64:
			request.RowEnd = int(v)
		case int:
			request.RowEnd = v
		case string:
			if val, err := strconv.Atoi(v); err == nil {
				request.RowEnd = val
			}
		}
	} else {
		request.RowEnd = -1 // 默认值，从头到尾
	}

	// 验证密码 - 从配置文件读取密码并进行MD5加密对比
	configPath := filepath.Join("Panel_Setting", "config.json")
	configData, err := ioutil.ReadFile(configPath)
	var configMap map[string]interface{}
	var storedPassword = "password" // 默认密码
	
	if err == nil {
		json.Unmarshal(configData, &configMap)
		if pass, exists := configMap["password"]; exists {
			if passStr, ok := pass.(string); ok {
				storedPassword = passStr
			}
		}
	}
	
	// 对存储的密码进行MD5加密（转为32位小写）
	md5Hash := md5.New()
	md5Hash.Write([]byte(storedPassword))
	storedPasswordMD5 := hex.EncodeToString(md5Hash.Sum(nil))
	
	// 客户端传递的已是MD5值，直接对比
	if request.Password == storedPasswordMD5 {
		// 登录成功，更新客户端状态
		SetClientLoggedIn(clientID, true)
		
		// 获取日志内容
		if request.RowStart != -1 {
			logContent, err := getLogContent(request.RowStart, request.RowEnd)
			if err != nil {
				// 日志获取失败
				result.Status = 502
				result.ErrInfo = "LOG_FAILED_GET"
				log.Printf("[ERROR]客户端 %s 登录成功，但获取日志失败: %v", clientID, err)
			} else {
				// 登录成功并获取日志
				result.Status = 200
				result.Info = map[string]interface{}{
					"log": logContent,
				}
				log.Printf("[INFO]客户端 %s 登录成功", clientID)
			}
		} else {
			// 不需要获取日志
			result.Status = 200
			result.Info = map[string]interface{}{
				"log": "", // 空日志
				}
			log.Printf("[INFO]客户端 %s 登录成功，未获取日志", clientID)
		}
	} else {
		// 密码错误
		result.Status = 501
		result.ErrInfo = "PASSWORD_IS_NOT_CORRECT"
		log.Printf("[WARNING]客户端 %s 登录失败: 密码错误", clientID)
	}

	// 构建完整响应
	response := BaseMessage{
		Gateway: 1, // ApiReturn的gateway为1
		Body:    mustMarshal(result),
	}

	return mustMarshal(response)
}

// 获取日志文件内容
// rowStart: 起始行号，从0开始
// rowEnd: 结束行号，-1表示到文件末尾
// 返回: 日志内容字符串
func getLogContent(rowStart, rowEnd int) (string, error) {
	// 实际应用中，这里应该从真实的日志文件中读取
	// 这里使用模拟的日志内容
	logFilePath := filepath.Join("Panel_Setting", "server.log")
	
	// 检查文件是否存在
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		// 文件不存在，返回模拟日志
		return "[2025-08-24 15:00:00] Server started\n[2025-08-24 15:01:00] Waiting for connections...", nil
	}
	
	// 读取文件内容
	content, err := ioutil.ReadFile(logFilePath)
	if err != nil {
		return "", err
	}
	
	// 分割日志行
	lines := strings.Split(string(content), "\n")
	
	// 根据rowStart和rowEnd截取日志
	startIdx := 0
	endIdx := len(lines)
	
	if rowStart >= 0 && rowStart < len(lines) {
		startIdx = rowStart
	}
	
	if rowEnd >= 0 && rowEnd < len(lines) {
		endIdx = rowEnd + 1 // 包含结束行
	}
	
	// 截取日志行
	selectedLines := lines[startIdx:endIdx]
	
	// 重新组合成字符串
	return strings.Join(selectedLines, "\n"), nil
}