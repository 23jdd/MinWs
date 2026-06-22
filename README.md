# MinWs

一个用 Go 标准库从零实现的轻量级 WebSocket 服务端，遵循 [RFC 6455](https://datatracker.ietf.org/doc/html/rfc6455)，**零第三方依赖**。

## 特性

- 完整的握手升级（`Sec-WebSocket-Accept` 计算 + 连接 Hijack）
- 帧解析与构建，支持 7/16/64 位负载长度
- 掩码（Masking）处理，原地异或
- 控制帧支持：`Ping` / `Pong` / `Close`
- 分片消息聚合（`Aggregator`），可配置最大负载上限
- 文本帧 UTF-8 校验、Close 状态码校验（RFC 6455 §7.4.1）
- 基于回调的事件模型：`OnOpen` / `OnMessage` / `OnClose` / `OnError`
- 使用 `sync.Pool` 复用读取缓冲，减少分配

## 环境要求

- Go 1.25+

## 快速开始

```go
package main

import "net/http"

func main() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		client, err := Upgrade(w, r)
		if err != nil {
			return
		}

		client.OnOpen = func() {
			// 连接建立
		}
		client.OnMessage = func(data []byte) {
			// 回显收到的消息
			client.SendText("echo: " + string(data))
		}
		client.OnClose = func() {
			// 连接关闭
		}
		client.OnError = func(err error) {
			// 处理错误
		}

		client.Listen() // 启动读取循环（非阻塞，内部 goroutine）
	})

	http.ListenAndServe(":8080", nil)
}
```

## API 概览

### 升级

| 函数 | 说明 |
| --- | --- |
| `Upgrade(w http.ResponseWriter, r *http.Request) (*Client, error)` | 完成握手并返回 `*Client` |

### Client

| 字段 / 方法 | 说明 |
| --- | --- |
| `OnOpen func()` | 连接建立时触发 |
| `OnMessage func(data []byte)` | 收到完整消息（文本/二进制）时触发 |
| `OnClose func()` | 连接关闭时触发 |
| `OnError func(err error)` | 发生错误时触发 |
| `Listen()` | 启动后台读取循环 |
| `SendText(text string)` | 发送文本帧 |
| `SendBinary(data []byte)` | 发送二进制帧 |
| `Ping()` | 发送 Ping 帧 |
| `Pong(payload []byte)` | 发送 Pong 帧 |
| `Close()` | 正常关闭连接 |
| `CloseWith(code uint16, reason string) error` | 以指定状态码和原因关闭 |

## 项目结构

| 文件 | 职责 |
| --- | --- |
| `Upgrader.go` | HTTP 握手升级与连接 Hijack |
| `FrameResolver.go` | 帧的读取、解析、构建与发送，Client 事件循环 |
| `Aggregation.go` | 分片消息聚合 |
| `Validation.go` | UTF-8 校验 |

## 许可证

[MIT](./LICENSE) © 2026 23jdd
