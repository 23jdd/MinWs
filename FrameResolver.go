package main

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"io"
	"sync"
)

// WebSocket 常量定义
const (
	OpcodeContinuation = 0x0
	OpcodeText         = 0x1
	OpcodeBinary       = 0x2
	OpcodeClose        = 0x8
	OpcodePing         = 0x9
	OpcodePong         = 0xA
)

type FrameHeader struct {
	Fin    bool
	Opcode byte
	Masked bool
	Length uint64
}

type Connect struct {
	c     *bufio.ReadWriter
	spool sync.Pool //    8 bytes
	bpool sync.Pool //   1024 bytes
	agg   *Aggregator
}

type Client struct {
	con       *Connect
	onOpen    func()
	onClose   func()
	onError   func(err error)
	onMessage func(data []byte)
}

func NewConnect(maxsize uint) *Connect {
	c := &Connect{}
	c.spool.New = func() interface{} {
		return make([]byte, 8)
	}
	c.bpool.New = func() interface{} {
		return make([]byte, 1024)
	}
	c.agg = NewAggregator(maxsize)
	return c
}

// Listen 持续监听并解析入站帧
func (c *Client) Listen() {
	go func() {
		if c.onOpen != nil {
			c.onOpen()
		}
		for {
			header, err := c.con.ReadHeader()
			if err != nil {
				if err != io.EOF {
					if c.onError != nil {
						c.onError(err)
					}
				}
				return
			}

			payload, err := c.con.ReadPayload(header)
			if err != nil {
				if c.onError != nil {
					c.onError(err)
				}
				continue
			}

			// 处理控制帧或分发消息
			switch header.Opcode {
			case OpcodePing:
				c.Pong(payload)
			case OpcodeClose:
				if c.onClose != nil {
					c.onClose()
				}
				return
			case OpcodeContinuation:
				if !c.con.agg.started {
					continue
				}
				if header.Fin {
					err := c.con.agg.Received(payload)
					c.con.agg.started = false
					if err != nil {
						if c.onError != nil {
							c.onError(err)
						}
						c.con.agg.Clear()
						continue
					}
					if c.con.agg.len == 0 {
						c.onMessage(nil)
					} else {
						c.onMessage(c.con.agg.buf[:c.con.agg.len])
					}
				} else {
					err := c.con.agg.Received(payload)
					if err != nil {
						if c.onError != nil {
							c.onError(err)
						}
						c.con.agg.Clear()
						c.con.agg.started = false
						continue
					}
				}
			case OpcodeText, OpcodeBinary:
				{
					if c.con.agg.started {
						continue
					}
					//  A frame
					if header.Fin {
						if c.onMessage != nil {
							c.onMessage(payload)
						}
						continue
					}
					err := c.con.agg.Received(payload)
					if err != nil {
						if c.onError != nil {
							c.onError(err)
						}
						c.con.agg.Clear()
						continue
					}
					c.con.agg.started = true
				}
			}
		}
	}()
}

// ReadHeader 解析报头
func (c *Connect) ReadHeader() (*FrameHeader, error) {
	pool := c.spool.Get()
	defer c.spool.Put(pool)
	buf := pool.([]byte)[:2]
	if _, err := io.ReadFull(c.c, buf); err != nil {
		return nil, err
	}
	header := &FrameHeader{
		Fin:    (buf[0] & 0x80) != 0,
		Opcode: buf[0] & 0x0F,
		Masked: (buf[1] & 0x80) != 0,
		Length: uint64(buf[1] & 0x7F),
	}
	// 处理长度扩展
	if header.Length == 126 {
		pool := c.spool.Get()
		defer c.spool.Put(pool)
		lenBuf := pool.([]byte)[:2] // TODO sync.pool
		if _, err := io.ReadFull(c.c, lenBuf); err != nil {
			return nil, err
		}
		header.Length = uint64(binary.BigEndian.Uint16(lenBuf))
	} else if header.Length == 127 {
		pool := c.spool.Get()
		defer c.spool.Put(pool)
		lenBuf := pool.([]byte) // TODO sync.pool
		if _, err := io.ReadFull(c.c, lenBuf); err != nil {
			return nil, err
		}
		header.Length = binary.BigEndian.Uint64(lenBuf)
	}
	return header, nil
}

// ReadPayload 读取并解密负载数据
func (c *Connect) ReadPayload(header *FrameHeader) ([]byte, error) {
	var maskKey []byte
	if header.Masked {
		pool := c.spool.Get()
		defer c.spool.Put(pool)
		maskKey = pool.([]byte)[:4] //sync pool
		if _, err := io.ReadFull(c.c, maskKey); err != nil {
			return nil, err
		}
	}

	payload := make([]byte, header.Length) //1024
	if _, err := io.ReadFull(c.c, payload); err != nil {

		return nil, err
	}

	if header.Masked {
		Mask(payload, maskKey)
	}

	return payload, nil
}

// WriteFrame 构建并发送 WebSocket 帧
func (c *Connect) WriteFrame(opcode byte, isMasked bool, data []byte) error {
	var header []byte
	length := len(data)
	// 1. 第一个字节 (Fin=1 + Opcode)
	header = append(header, 0x80|opcode)

	// 2. 长度和 Mask 标志位
	maskBit := byte(0)
	if isMasked {
		maskBit = 0x80
	}

	if length <= 125 {
		header = append(header, maskBit|byte(length))
	} else if length <= 65535 {
		header = append(header, maskBit|126)
		extLen := make([]byte, 2)
		binary.BigEndian.PutUint16(extLen, uint16(length))
		header = append(header, extLen...)
	} else {
		header = append(header, maskBit|127)
		extLen := make([]byte, 8)
		binary.BigEndian.PutUint64(extLen, uint64(length))
		header = append(header, extLen...)
	}

	// 3. 处理 Masking Key 和加密
	if isMasked {
		maskKey := make([]byte, 4)
		if _, err := io.ReadFull(rand.Reader, maskKey); err != nil {
			return err
		}
		header = append(header, maskKey...)

		// 为了不破坏原数据，拷贝一份进行处理
		maskedData := make([]byte, length)
		copy(maskedData, data)
		Mask(maskedData, maskKey)
		data = maskedData
	}

	// 4. 发送 Header + Payload
	if _, err := c.c.Write(header); err != nil {
		return err
	}
	if _, err := c.c.Write(data); err != nil {
		return err
	}
	return c.c.Flush()
}

// Mask 统一掩码算法 (原地异或，高效)
func Mask(data []byte, key []byte) {
	for i := 0; i < len(data); i++ {
		data[i] ^= key[i%4]
	}
}

// --- Client 暴露的方法 ---

func (client *Client) SendText(text string) {
	_ = client.con.WriteFrame(OpcodeText, false, []byte(text))
}

func (client *Client) SendBinary(data []byte) {
	_ = client.con.WriteFrame(OpcodeBinary, false, data)
}

func (client *Client) Ping() {
	_ = client.con.WriteFrame(OpcodePing, false, nil)
}

func (client *Client) Pong(pingPayload []byte) {
	_ = client.con.WriteFrame(OpcodePong, true, pingPayload)
}

func (client *Client) Close() {
	_ = client.con.WriteFrame(OpcodeClose, true, nil)
}
