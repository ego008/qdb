package zset

import (
	"bytes"
	"encoding/binary"

	"qdb/backend"
	"qdb/pkg/encoding"
	"qdb/pkg/lock"
)

const (
	ZSetDataPrefix  byte = 0x02 // Data: [Prefix][Name][Member] -> Score
	ZSetIndexPrefix byte = 0x03 // Index: [Prefix][Name][Score][Member] -> Nil
)

type ZSetEngine struct {
	be     backend.Backend
	locker *lock.Locker
}

func NewZSetEngine(be backend.Backend) *ZSetEngine {
	return &ZSetEngine{
		be:     be,
		locker: lock.NewLocker(64),
	}
}

func (z *ZSetEngine) ZSet(name string, member []byte, score uint64) error {
	z.locker.Lock(name)
	defer z.locker.Unlock(name)

	dataKey := encoding.EncodeKey(ZSetDataPrefix, name, member)

	oldVal, err := z.be.Get(dataKey)
	batch := z.be.NewBatch()
	defer batch.Close()

	if err == nil && len(oldVal) == 8 {
		oldScore := binary.BigEndian.Uint64(oldVal)
		if oldScore == score {
			return nil
		}

		oldScoreBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(oldScoreBuf, oldScore)
		oldIndexKey := encoding.EncodeKey(ZSetIndexPrefix, name, append(oldScoreBuf, member...))
		batch.Delete(oldIndexKey)
	}

	newScoreBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(newScoreBuf, score)

	batch.Put(dataKey, newScoreBuf)

	newIndexKey := encoding.EncodeKey(ZSetIndexPrefix, name, append(newScoreBuf, member...))
	batch.Put(newIndexKey, []byte{})

	return batch.Commit()
}

func (z *ZSetEngine) ZGet(name string, member []byte) (uint64, error) {
	z.locker.RLock(name)
	defer z.locker.RUnlock(name)

	dataKey := encoding.EncodeKey(ZSetDataPrefix, name, member)
	val, err := z.be.Get(dataKey)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(val), nil
}

func (z *ZSetEngine) ZScan(name string, memberStart []byte, scoreStart uint64, limit int) ([]interface{}, error) {
	z.locker.RLock(name)
	defer z.locker.RUnlock(name)

	prefix := encoding.EncodePrefix(ZSetIndexPrefix, name)

	scoreBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(scoreBuf, scoreStart)
	startPayload := append(scoreBuf, memberStart...)
	targetStart := encoding.EncodeKey(ZSetIndexPrefix, name, startPayload)

	iter := z.be.NewIterator(prefix)
	defer iter.Close()

	var result []interface{}
	count := 0

	for ok := iter.Seek(targetStart); ok; ok = iter.Next() {
		currKey := iter.Key()

		if !bytes.HasPrefix(currKey, prefix) {
			break
		}

		if len(memberStart) > 0 && bytes.Equal(currKey, targetStart) {
			continue
		}

		// 精确截取剥离了 Name 之后的部分：[Score(8B) + Member]
		payload := extractIndexPayload(currKey, name)
		if len(payload) < 8 {
			continue
		}

		score := binary.BigEndian.Uint64(payload[:8])
		member := append([]byte(nil), payload[8:]...)

		result = append(result, string(member), score)
		count++

		if limit > 0 && count >= limit {
			break
		}
	}

	return result, iter.Error()
}

func (z *ZSetEngine) ZRScan(name string, memberStart []byte, scoreStart uint64, limit int) ([]interface{}, error) {
	z.locker.RLock(name)
	defer z.locker.RUnlock(name)

	prefix := encoding.EncodePrefix(ZSetIndexPrefix, name)
	iter := z.be.NewIterator(prefix)
	defer iter.Close()

	var result []interface{}
	count := 0

	var ok bool
	if len(memberStart) > 0 || scoreStart > 0 {
		scoreBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(scoreBuf, scoreStart)
		startPayload := append(scoreBuf, memberStart...)
		targetStart := encoding.EncodeKey(ZSetIndexPrefix, name, startPayload)

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

		payload := extractIndexPayload(currKey, name)
		if len(payload) < 8 {
			continue
		}

		score := binary.BigEndian.Uint64(payload[:8])
		member := append([]byte(nil), payload[8:]...)

		result = append(result, string(member), score)
		count++

		if limit > 0 && count >= limit {
			break
		}
	}

	return result, iter.Error()
}

// 辅助函数：安全计算并剥离 Header 前缀，获取 [Score(8B) + Member] Payload
func extractIndexPayload(encoded []byte, name string) []byte {
	// 格式: 1B(Prefix) + Varint(NameLen) + Name + Score(8B) + Member
	_, n := binary.Uvarint(encoded[1:])
	headerLen := 1 + n + len(name)
	return encoded[headerLen:]
}
