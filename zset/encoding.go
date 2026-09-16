package zset

import (
	"bytes"
	"encoding/binary"
)

var (
	HashPrefix     = []byte{30}
	ZetKeyPrefix   = []byte{31}
	ZetScorePrefix = []byte{29}
	SplitChar      = []byte{28}
)

func EncodeUint64(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func DecodeUint64(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b[:8])
}

func Concat(slices ...[]byte) []byte {
	return bytes.Join(slices, nil)
}
