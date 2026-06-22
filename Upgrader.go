package MinWs

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

func computeAcceptKey(clientKey string) string {
	const magicString = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	fullString := clientKey + magicString

	h := sha1.New()
	h.Write([]byte(fullString))
	sha1Sum := h.Sum(nil)

	return base64.StdEncoding.EncodeToString(sha1Sum)
}
func Upgrade(w http.ResponseWriter, r *http.Request) (*Client, error) {
	connection := r.Header.Get("Connection")
	if !strings.Contains(connection, "Upgrade") {
		return nil, fmt.Errorf("upgrade not supported") //
	}
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return nil, fmt.Errorf("upgrade header must be websocket")
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		w.Header().Set("Sec-WebSocket-Version", "13")
		http.Error(w, "Upgrade Required", http.StatusUpgradeRequired)
		return nil, fmt.Errorf("unsupported websocket version")
	}
	if r.Header.Get("Sec-WebSocket-Key") == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return nil, fmt.Errorf("missing Sec-WebSocket-Key")
	}
	hijacker, ok := w.(http.Hijacker)
	if ok {
		_, brw, err := hijacker.Hijack()
		if err != nil {
			panic(err)
		}

		acceptKey := computeAcceptKey(r.Header.Get("Sec-WebSocket-Key"))

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
			return nil, err
		}
		err = brw.Flush()
		if err != nil {
			return nil, err
		}
		connect := NewConnect(1024)
		connect.c = brw
		return &Client{
			con: connect,
		}, nil
	} else {
		return nil, fmt.Errorf("hijacker not supported")
	}
}
