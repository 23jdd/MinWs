package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

type FrameHeader struct {
	Fin    bool  // 0-1
	Opcode byte  //
	Masked bool  //
	Length int64 //
}
type FramePreLoad struct {
}
type Client struct {
	con       Connect
	onOpen    func()
	onClose   func()
	onError   func(err error)
	onMessage func(data []byte)
}
type Connect struct {
	c *bufio.ReadWriter
}

func (c *Client) Listen() {
	go func() {
		for {
			header, err := c.con.ReadHeader()
			fmt.Println("Read header:", header)
			if err != nil {
				c.onError(err)
				return
			}
			err = c.con.ReadPreLoad(header)
			if err != nil {
				c.onError(err)
			}
		}

	}()
}
func Decode(mask uint32, data []byte) []byte {
	result := make([]byte, len(data))
	for index, v := range data {
		switch index % 4 {
		case 0:
			result[index] = v ^ byte(mask>>24)
		case 1:
			result[index] = v ^ byte(mask>>16)
		case 2:
			result[index] = v ^ byte(mask>>8)
		case 3:
			result[index] = v ^ byte(mask)
		}
	}
	return result
}

type Frame struct {
	header FrameHeader
}

func (c *Client) Connect() error {
	return nil
}

// 0000 0000 0000 0000
func NewFrameHeader(data uint16) FrameHeader {
	fin := data & 0x8000
	header := FrameHeader{}
	if fin > 0 {
		header.Fin = true
	}
	//  0000 0000 0000 0000
	header.Opcode = byte((data & 0x0F00) >> 8)
	header.Masked = true
	// 1 7
	header.Length = int64(data & 0x007F)
	return header
}
func (c *Connect) ReadHeader() (*FrameHeader, error) {
	buffer := make([]byte, 2) //TODO  sync.pool
	_, err := io.ReadFull(c.c, buffer)
	if err != nil {
		return nil, err
	}
	frameHeader := NewFrameHeader(binary.BigEndian.Uint16(buffer))
	return &frameHeader, nil
}
func (c *Connect) ReadPreLoad(header *FrameHeader) error {
	b_4 := make([]byte, 4)
	_, err := io.ReadFull(c.c, b_4)
	if err != nil {
		return err
	}
	buffer := make([]byte, header.Length)
	_, err = io.ReadFull(c.c, buffer)
	if err != nil {
		return err
	}
	decode := Decode(binary.BigEndian.Uint32(b_4), buffer)
	fmt.Println(string(decode))
	return nil
}
