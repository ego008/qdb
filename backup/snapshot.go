package backup

import (
	"encoding/binary"
	"io"
	"os"

	"qdb/backend"
)

// ExportSnapshot 将后端的全量 Key-Value 导出至 Writer
func ExportSnapshot(be backend.Backend, w io.Writer) error {
	iter := be.NewIterator(nil)
	defer iter.Close()

	for ok := iter.First(); ok; ok = iter.Next() {
		k := iter.Key()
		v := iter.Value()

		// 格式: [KeyLen uint32][ValLen uint32][Key bytes][Val bytes]
		if err := binary.Write(w, binary.BigEndian, uint32(len(k))); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, uint32(len(v))); err != nil {
			return err
		}
		if _, err := w.Write(k); err != nil {
			return err
		}
		if _, err := w.Write(v); err != nil {
			return err
		}
	}
	return iter.Error()
}

// ImportSnapshot 从 Reader 读取数据全量恢复至后端
func ImportSnapshot(be backend.Backend, r io.Reader) error {
	batch := be.NewBatch()
	defer batch.Close()

	for {
		var kLen, vLen uint32
		if err := binary.Read(r, binary.BigEndian, &kLen); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := binary.Read(r, binary.BigEndian, &vLen); err != nil {
			return err
		}

		k := make([]byte, kLen)
		if _, err := io.ReadFull(r, k); err != nil {
			return err
		}

		v := make([]byte, vLen)
		if _, err := io.ReadFull(r, v); err != nil {
			return err
		}

		// 修复：batch.Put 不返回值，单独调用后再统一 Batch.Commit() 校验错误
		batch.Put(k, v)
	}

	return batch.Commit()
}

// ExportToFile 保存快照至文件
func ExportToFile(be backend.Backend, filepath string) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	return ExportSnapshot(be, f)
}

// ImportFromFile 从文件恢复快照
func ImportFromFile(be backend.Backend, filepath string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	return ImportSnapshot(be, f)
}
