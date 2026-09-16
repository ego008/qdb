package encoding

import (
	"encoding/binary"
	"math"
)

// EncodeKey 将 [type, name, key] 安全打包，避免分隔符冲突
func EncodeKey(prefix byte, name string, key []byte) []byte {
	nameBytes := []byte(name)
	nameLen := len(nameBytes)
	keyLen := len(key)

	buf := make([]byte, 1+binary.MaxVarintLen64+nameLen+keyLen)
	buf[0] = prefix

	n := binary.PutUvarint(buf[1:], uint64(nameLen))
	idx := 1 + n

	copy(buf[idx:], nameBytes)
	idx += nameLen

	copy(buf[idx:], key)
	return buf[:idx+keyLen]
}

// EncodePrefix 解析 Key 的 Prefix 前缀（用于 Scan 前缀匹配）
func EncodePrefix(prefix byte, name string) []byte {
	nameBytes := []byte(name)
	buf := make([]byte, 1+binary.MaxVarintLen64+len(nameBytes))
	buf[0] = prefix

	n := binary.PutUvarint(buf[1:], uint64(len(nameBytes)))
	idx := 1 + n

	copy(buf[idx:], nameBytes)
	return buf[:idx]
}

// EncodeFloat64ToSortableBytes 将 float64 编码为保持大小字典序的 8 字节
func EncodeFloat64ToSortableBytes(val float64) []byte {
	bits := math.Float64bits(val)
	if val >= 0 {
		bits ^= 0x8000000000000000 // 正数翻转符号位
	} else {
		bits = ^bits // 负数按位取反
	}
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, bits)
	return buf
}

// DecodeSortableBytesToFloat64 将 8 字节字节序解码回 float64
func DecodeSortableBytesToFloat64(b []byte) float64 {
	bits := binary.BigEndian.Uint64(b)
	if bits&0x8000000000000000 != 0 {
		bits ^= 0x8000000000000000
	} else {
		bits = ^bits
	}
	return math.Float64frombits(bits)
}
