# Zephyr-Craft-Panel-2-For-BDS

使用golang进行开发，针对 **[Bedrock Dedicated Server](https://www.minecraft.net/zh-hans/download/server/bedrock)** 进行适配的服务端

此仓库的主要介绍文档在 ZCP2的 **[主仓库](https://www.github.com/ZCP2-DEV/zephyr-craft-panel-2)** ，如有需要可以前往

## 编译步骤

### 环境要求
- 安装 [Go 1.21+](https://golang.org/dl/) 开发环境
- 配置好 `GOPATH` 和 `GOROOT` 环境变量
- 确保网络通畅（需下载依赖包）


### 编译前准备
```bash
# 克隆仓库
git clone https://github.com/ZephyrCraft-Panel-2/Server-BDS.git
cd Server-BDS
```

### 依赖安装
```bash
# 下载项目依赖
go mod download
```
### 本地编译
```bash
# 编译生成可执行文件
go build -o Server-BDS.exe
```
