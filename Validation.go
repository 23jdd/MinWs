package main

import "unicode/utf8"

func IsValidUtf8(d []byte) bool {
	return utf8.Valid(d)
}

type Validator struct {
}
