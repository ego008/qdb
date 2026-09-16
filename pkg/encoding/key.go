package encoding

import (
	"encoding/binary"
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
