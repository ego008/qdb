package hash

import (
	"bytes"
	"encoding/binary"

	"qdb/backend"
	"qdb/pkg/encoding"
	"qdb/pkg/lock"
)

const HashPrefix byte = 0x01

type HashEngine struct {
	be     backend.Backend
	locker *lock.Locker
}

func NewHashEngine(be backend.Backend) *HashEngine {
	return &HashEngine{
		be:     be,
		locker: lock.NewLocker(64),
	}
}

func (h *HashEngine) HSet(name string, key, value []byte) error {
	h.locker.Lock(name)
	defer h.locker.Unlock(name)

	encKey := encoding.EncodeKey(HashPrefix, name, key)
	return h.be.Put(encKey, value)
}

func (h *HashEngine) HGet(name string, key []byte) ([]byte, error) {
	h.locker.RLock(name)
	defer h.locker.RUnlock(name)

	encKey := encoding.EncodeKey(HashPrefix, name, key)
	return h.be.Get(encKey)
}

func (h *HashEngine) HScan(name string, keyStart []byte, limit int) ([][]byte, error) {
	h.locker.RLock(name)
	defer h.locker.RUnlock(name)

	prefix := encoding.EncodePrefix(HashPrefix, name)
	targetStart := encoding.EncodeKey(HashPrefix, name, keyStart)

	iter := h.be.NewIterator(prefix)
	defer iter.Close()

	var result [][]byte
	count := 0

	for ok := iter.Seek(targetStart); ok; ok = iter.Next() {
		currKey := iter.Key()

		if !bytes.HasPrefix(currKey, prefix) {
			break
		}

		if len(keyStart) > 0 && bytes.Equal(currKey, targetStart) {
			continue
		}

		// 精确剥离 [Prefix(1B) + Varint(NameLen) + Name]
		realKey := extractRealKey(currKey, name)
		val := append([]byte(nil), iter.Value()...)

		result = append(result, realKey, val)
		count++

		if limit > 0 && count >= limit {
			break
		}
	}

	return result, iter.Error()
}

func (h *HashEngine) HRScan(name string, keyStart []byte, limit int) ([][]byte, error) {
	h.locker.RLock(name)
	defer h.locker.RUnlock(name)

	prefix := encoding.EncodePrefix(HashPrefix, name)
	iter := h.be.NewIterator(prefix)
	defer iter.Close()

	var result [][]byte
	count := 0

	var ok bool
	if len(keyStart) > 0 {
		targetStart := encoding.EncodeKey(HashPrefix, name, keyStart)
		ok = iter.Seek(targetStart)
		if !ok {
			ok = iter.Last()
		} else {
			ok = iter.Prev()
		}
	} else {
		ok = iter.Last()
	}

	for ; ok; ok = iter.Prev() {
		currKey := iter.Key()

		if !bytes.HasPrefix(currKey, prefix) {
			break
		}

		realKey := extractRealKey(currKey, name)
		val := append([]byte(nil), iter.Value()...)

		result = append(result, realKey, val)
		count++

		if limit > 0 && count >= limit {
			break
		}
	}

	return result, iter.Error()
}

// 辅助函数：安全计算并剥离 Header 前缀
func extractRealKey(encoded []byte, name string) []byte {
	// 格式: 1B(Prefix) + Varint(NameLen) + Name + Key
	_, n := binary.Uvarint(encoded[1:])
	headerLen := 1 + n + len(name)
	return append([]byte(nil), encoded[headerLen:]...)
}
