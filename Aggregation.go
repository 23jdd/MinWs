package main

import (
	"errors"
)

type Aggregator struct {
	len     int
	buf     []byte
	started bool
	opcode  byte
}

func NewAggregator(maxsize uint) *Aggregator {
	a := Aggregator{}
	a.buf = make([]byte, maxsize)
	return &a
}
func (a *Aggregator) Received(data []byte) error {
	temp := a.len + len(data)
	if temp > len(a.buf) {
		return errors.New("too many bytes")
	}
	for _, v := range data {
		a.buf[a.len] = v
		a.len++
	}
	return nil
}
func (a *Aggregator) Clear() {
	a.len = 0
	a.started = false
	a.opcode = 0
}
