package MinWs

import "unicode/utf8"

func IsValidUtf8(d []byte) bool {
	return utf8.Valid(d)
}
