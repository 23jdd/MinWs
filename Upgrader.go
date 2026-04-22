package main

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// [1 0 0 0 0 0 0 1 ]
// [1 1 0 1 0 0 0 1 ] 1+
// real [1 0 0 0 0 0 0 1] 1 1 0 1 0 0 0 1
func BitFormat(b byte) string {
	var bits [8]string
	for i := 0; i < 8; i++ {
		// Check bit from MSB (7) to LSB (0)
		if b&(1<<(7-i)) != 0 {
			bits[i] = "1"
		} else {
			bits[i] = "0"
		}
	}
	return fmt.Sprintf("b7 b6 b5 b4 b3 b2 b1 b0\n%s  %s  %s  %s  %s  %s  %s  %s",
		bits[0], bits[1], bits[2], bits[3], bits[4], bits[5], bits[6], bits[7])
}
func computeAcceptKey(clientKey string) string {
	// 1. 拼接
	const magicString = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	fullString := clientKey + magicString

	// 2. SHA-1 哈希
	h := sha1.New()
	h.Write([]byte(fullString))
	sha1Sum := h.Sum(nil)

	// 3. Base64 编码
	return base64.StdEncoding.EncodeToString(sha1Sum)
}
func Upgrade(w http.ResponseWriter, r *http.Request) (bool, error) {
	connection := r.Header.Get("Connection")
	if !strings.Contains(connection, "Upgrade") {
		return false, fmt.Errorf("upgrade not supported") //
	}
	hijacker, ok := w.(http.Hijacker)
	if ok {
		_, brw, err := hijacker.Hijack()
		if err != nil {
			panic(err)
		}
		// 假设你已经通过 Hijack 拿到了 brw (*bufio.ReadWriter)
		acceptKey := computeAcceptKey(r.Header.Get("Sec-WebSocket-Key"))

		// 构造响应头
		response := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + acceptKey + "\r\n"

		// 如果有子协议支持（可选）
		if subProto := r.Header.Get("Sec-WebSocket-Protocol"); subProto != "" {
			response += "Sec-WebSocket-Protocol: " + subProto + "\r\n"
		}

		response += "\r\n" // 最后必须有一个空行表示 Header 结束

		// 写入并 Flush
		_, err = brw.WriteString(response)
		if err != nil {
			return false, err
		}
		err = brw.Flush()
		if err != nil {
			return false, err
		}
		connect := Connect{c: brw}
		client := Client{
			con: connect,
			onOpen: func() {

			},
			onClose: func() {

			},
			onError: func(err error) {
				fmt.Println(err)
			},
			onMessage: func(data []byte) {

			},
		}
		client.Listen()
		return true, nil
	} else {
		return false, fmt.Errorf("hijacker not supported")
	}
}
