package zset

import (
	"bytes"
	"encoding/binary"

	"qdb/backend"
	"qdb/pkg/encoding"
	"qdb/pkg/lock"
)

const (
	ZSetDataPrefix  byte = 0x02 // Data: [Prefix][Name][Member] -> Score(8B Sortable)
	ZSetIndexPrefix byte = 0x03 // Index: [Prefix][Name][Score(8B Sortable)][Member] -> Nil
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

func (z *ZSetEngine) ZSet(name string, member []byte, score float64) error {
	z.locker.Lock(name)
	defer z.locker.Unlock(name)

	dataKey := encoding.EncodeKey(ZSetDataPrefix, name, member)

	oldVal, err := z.be.Get(dataKey)
	batch := z.be.NewBatch()
	defer batch.Close()

	if err == nil && len(oldVal) == 8 {
		oldScore := encoding.DecodeSortableBytesToFloat64(oldVal)
		if oldScore == score {
			return nil
		}

		oldScoreBuf := encoding.EncodeFloat64ToSortableBytes(oldScore)
		oldIndexKey := encoding.EncodeKey(ZSetIndexPrefix, name, append(oldScoreBuf, member...))
		batch.Delete(oldIndexKey)
	}

	newScoreBuf := encoding.EncodeFloat64ToSortableBytes(score)
	batch.Put(dataKey, newScoreBuf)

	newIndexKey := encoding.EncodeKey(ZSetIndexPrefix, name, append(newScoreBuf, member...))
	batch.Put(newIndexKey, []byte{})

	return batch.Commit()
}

func (z *ZSetEngine) ZGet(name string, member []byte) (float64, error) {
	z.locker.RLock(name)
	defer z.locker.RUnlock(name)

	dataKey := encoding.EncodeKey(ZSetDataPrefix, name, member)
	val, err := z.be.Get(dataKey)
	if err != nil {
		return 0, err
	}
	return encoding.DecodeSortableBytesToFloat64(val), nil
}

func (z *ZSetEngine) ZRem(name string, member []byte) (bool, error) {
	z.locker.Lock(name)
	defer z.locker.Unlock(name)

	dataKey := encoding.EncodeKey(ZSetDataPrefix, name, member)
	val, err := z.be.Get(dataKey)
	if err != nil {
		return false, nil
	}

	batch := z.be.NewBatch()
	defer batch.Close()

	batch.Delete(dataKey)

	indexKey := encoding.EncodeKey(ZSetIndexPrefix, name, append(val, member...))
	batch.Delete(indexKey)

	err = batch.Commit()
	return err == nil, err
}

func (z *ZSetEngine) ZRank(name string, member []byte) (int64, error) {
	z.locker.RLock(name)
	defer z.locker.RUnlock(name)

	targetScore, err := z.ZGet(name, member)
	if err != nil {
		return -1, err
	}

	prefix := encoding.EncodePrefix(ZSetIndexPrefix, name)
	iter := z.be.NewIterator(prefix)
	defer iter.Close()

	var rank int64 = 0
	targetScoreBuf := encoding.EncodeFloat64ToSortableBytes(targetScore)
	targetIndexKey := encoding.EncodeKey(ZSetIndexPrefix, name, append(targetScoreBuf, member...))

	for ok := iter.Seek(prefix); ok; ok = iter.Next() {
		currKey := iter.Key()
		if !bytes.HasPrefix(currKey, prefix) {
			break
		}
		if bytes.Equal(currKey, targetIndexKey) {
			return rank, nil
		}
		rank++
	}
	return -1, nil
}

func (z *ZSetEngine) ZCount(name string, minScore, maxScore float64) (int64, error) {
	z.locker.RLock(name)
	defer z.locker.RUnlock(name)

	prefix := encoding.EncodePrefix(ZSetIndexPrefix, name)
	minBuf := encoding.EncodeFloat64ToSortableBytes(minScore)
	startKey := encoding.EncodeKey(ZSetIndexPrefix, name, minBuf)

	iter := z.be.NewIterator(prefix)
	defer iter.Close()

	var count int64 = 0
	for ok := iter.Seek(startKey); ok; ok = iter.Next() {
		currKey := iter.Key()
		if !bytes.HasPrefix(currKey, prefix) {
			break
		}

		payload := extractIndexPayload(currKey, name)
		if len(payload) < 8 {
			continue
		}

		score := encoding.DecodeSortableBytesToFloat64(payload[:8])
		if score > maxScore {
			break
		}
		count++
	}
	return count, iter.Error()
}

func (z *ZSetEngine) ZScan(name string, memberStart []byte, scoreStart float64, limit int) ([]interface{}, error) {
	z.locker.RLock(name)
	defer z.locker.RUnlock(name)

	prefix := encoding.EncodePrefix(ZSetIndexPrefix, name)

	scoreBuf := encoding.EncodeFloat64ToSortableBytes(scoreStart)
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

		payload := extractIndexPayload(currKey, name)
		if len(payload) < 8 {
			continue
		}

		score := encoding.DecodeSortableBytesToFloat64(payload[:8])
		member := append([]byte(nil), payload[8:]...)

		result = append(result, string(member), score)
		count++

		if limit > 0 && count >= limit {
			break
		}
	}

	return result, iter.Error()
}

func (z *ZSetEngine) ZRScan(name string, memberStart []byte, scoreStart float64, limit int) ([]interface{}, error) {
	z.locker.RLock(name)
	defer z.locker.RUnlock(name)

	prefix := encoding.EncodePrefix(ZSetIndexPrefix, name)
	iter := z.be.NewIterator(prefix)
	defer iter.Close()

	var result []interface{}
	count := 0

	var ok bool
	if len(memberStart) > 0 || scoreStart != 0 {
		scoreBuf := encoding.EncodeFloat64ToSortableBytes(scoreStart)
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

		score := encoding.DecodeSortableBytesToFloat64(payload[:8])
		member := append([]byte(nil), payload[8:]...)

		result = append(result, string(member), score)
		count++

		if limit > 0 && count >= limit {
			break
		}
	}

	return result, iter.Error()
}

func extractIndexPayload(encoded []byte, name string) []byte {
	_, n := binary.Uvarint(encoded[1:])
	headerLen := 1 + n + len(name)
	return encoded[headerLen:]
}
